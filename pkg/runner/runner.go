package runner

import (
	"context"
	"fmt"
	"resticontainer/pkg/apptypes"
	"resticontainer/pkg/dockerclient"
)

type ResticExecutor func(ctx context.Context, args []string) error

func RunHooksAndBackup(ctx context.Context, client dockerclient.DockerClient, targets []*apptypes.Target, resticArgs []string, execRestic ResticExecutor) error {
	var executedPreHooks []*apptypes.Target

	// Pre hooks
	for _, t := range targets {
		if t.PreHook != "" {
			cmd := []string{"sh", "-c", t.PreHook}
			err := client.ExecCommand(ctx, t.ContainerID, cmd)
			if err != nil {
				return fmt.Errorf("pre-hook failed for %s: %w", t.ContainerID, err)
			}
		}
		executedPreHooks = append(executedPreHooks, t)
	}

	// Make sure post-hooks run for all successfully executed pre-hooks
	defer func() {
		for _, t := range executedPreHooks {
			if t.PostHook != "" {
				cmd := []string{"sh", "-c", t.PostHook}
				err := client.ExecCommand(ctx, t.ContainerID, cmd)
				if err != nil {
					fmt.Printf("Warning: post-hook failed for %s: %v\n", t.ContainerID, err)
				}
			}
		}
	}()

	var backupPaths []string
	for _, t := range targets {
		backupPaths = append(backupPaths, t.Paths...)
	}

	if len(backupPaths) > 0 {
		args := append([]string{"backup"}, backupPaths...)
		args = append(args, resticArgs...)
		err := execRestic(ctx, args)
		if err != nil {
			return fmt.Errorf("restic backup failed: %w", err)
		}
	}

	return nil
}
