package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"resticontainer/pkg/apptypes"
	"resticontainer/pkg/discovery"
	"resticontainer/pkg/dockerclient"
	"resticontainer/pkg/runner"
)

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

	resticArgs := os.Args[2:]
	err = runner.RunHooksAndBackup(ctx, client, targets, resticArgs, runResticNative)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
		os.Exit(1)
	}
}
