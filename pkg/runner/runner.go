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
	var stoppedContainers []*apptypes.Target

	defer func() {
		for _, t := range stoppedContainers {
			if err := client.StartContainer(context.Background(), t.ContainerID); err != nil {
				fmt.Printf("Warning: failed to start container %s after backup: %v\n", t.ContainerID, err)
			}
		}

		for _, t := range executedPreHooks {
			if t.PostHook != "" {
				cmd := []string{"sh", "-c", t.PostHook}
				err := client.ExecCommand(context.Background(), t.ContainerID, cmd)
				if err != nil {
					fmt.Printf("Warning: post-hook failed for %s: %v\n", t.ContainerID, err)
				}
			}
		}
	}()

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

	// Stop marked containers
	for _, t := range targets {
		if t.Stop {
			if err := client.StopContainer(ctx, t.ContainerID); err != nil {
				return fmt.Errorf("failed to stop container %s: %w", t.ContainerID, err)
			}
			stoppedContainers = append(stoppedContainers, t)
		}
	}

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
