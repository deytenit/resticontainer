package runner

import (
	"context"
	"errors"
	"testing"

	"resticontainer/pkg/apptypes"

	"github.com/stretchr/testify/assert"
)

type failingHookClient struct {
	mockDockerClient
	err error
}

func (c *failingHookClient) ExecCommand(ctx context.Context, containerID string, cmd []string) error {
	return c.err
}

func TestRunHooksAndBackup_PreHookFailurePreventsBackup(t *testing.T) {
	hookErr := errors.New("exec failed with exit code 23")
	client := &failingHookClient{err: hookErr}
	targets := []*apptypes.Target{{ContainerID: "database", PreHook: "dump database", Stop: true}}
	backupCalled := false

	err := RunHooksAndBackup(context.Background(), client, targets, []string{"/dumps"}, nil, func(context.Context, []string) error {
		backupCalled = true
		return nil
	})

	assert.ErrorIs(t, err, hookErr)
	assert.ErrorContains(t, err, "pre-hook failed for database")
	assert.False(t, backupCalled)
	assert.Empty(t, client.stops)
	assert.Empty(t, client.starts)
}
