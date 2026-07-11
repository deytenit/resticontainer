package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	"resticontainer/pkg/apptypes"
	"resticontainer/pkg/discovery"
	"resticontainer/pkg/dockerclient"
	"resticontainer/pkg/lockfile"
	"resticontainer/pkg/runner"
)

const defaultLockPath = "/var/lib/resticontainer/restic-backup-lock.json"

func runResticNative(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "restic", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func main() {
	if len(os.Args) < 2 {
		if err := runResticNative(context.Background(), os.Args[1:]); err != nil {
			os.Exit(1)
		}
		return
	}

	command := os.Args[1]
	if command != "backup" {
		if err := runResticNative(context.Background(), os.Args[1:]); err != nil {
			os.Exit(1)
		}
		return
	}

	ctx := context.Background()
	client, err := dockerclient.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize docker client: %v\n", err)
		os.Exit(1)
	}

	hostfs := os.Getenv("RESTIC_HOSTFS")
	if hostfs == "" {
		hostfs = "/hostfs"
	}

	lockPath := os.Getenv("RESTICONTAINER_LOCK")
	if lockPath == "" {
		lockPath = defaultLockPath
	}

	lock, err := lockfile.Load(lockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read lock file %s, continuing without it: %v\n", lockPath, err)
	}

	containers, err := client.ListContainers(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list containers: %v\n", err)
		os.Exit(1)
	}

	var targets []*apptypes.Target
	for _, c := range containers {
		inspect, err := client.InspectContainer(ctx, c.ID)
		if err != nil {
			continue
		}
		target, _ := discovery.ParseContainer(inspect, hostfs)
		if target != nil {
			targets = append(targets, target)
		}
	}

	// Refresh the lock file from the containers discovered this run: remember paths for
	// containers that opt in, forget those that opt out. Persist before the backup so
	// the state survives even if the backup itself fails.
	now := time.Now().UTC()
	for _, t := range targets {
		if t.Lock {
			lock.Upsert(t.ContainerID, t.Name, t.Paths, now)
		} else {
			lock.Delete(t.ContainerID)
		}
	}
	if err := lockfile.Save(lockPath, lock); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write lock file %s: %v\n", lockPath, err)
	}

	backupPaths := mergeBackupPaths(targets, lock)

	resticArgs := os.Args[2:]
	err = runner.RunHooksAndBackup(ctx, client, targets, backupPaths, resticArgs, runResticNative)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
		os.Exit(1)
	}
}

// mergeBackupPaths unions the freshly resolved paths of every running target (so a
// container with restic.backup.lock=false is still backed up while running) with the
// existence-filtered paths remembered in the lock file (covering containers that are
// down or removed), deduped and stably ordered.
func mergeBackupPaths(targets []*apptypes.Target, lock *lockfile.LockFile) []string {
	set := map[string]struct{}{}
	for _, t := range targets {
		for _, p := range t.Paths {
			set[p] = struct{}{}
		}
	}
	for _, p := range lock.BackupPaths(pathExists) {
		set[p] = struct{}{}
	}
	paths := make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
