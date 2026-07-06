package runner

import (
	"context"
	"fmt"
	"testing"
	"resticontainer/pkg/apptypes"
	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
)

type execCall struct {
	containerID string
	cmd         []string
}

type mockDockerClient struct {
	execs []execCall
}

func (m *mockDockerClient) ListContainers(ctx context.Context) ([]types.Container, error) {
	return nil, nil
}
func (m *mockDockerClient) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}
func (m *mockDockerClient) ExecCommand(ctx context.Context, containerID string, cmd []string) error {
	m.execs = append(m.execs, execCall{containerID: containerID, cmd: cmd})
	return nil
}

func TestRunHooksAndBackup(t *testing.T) {
	client := &mockDockerClient{}
	targets := []*apptypes.Target{
		{
			ContainerID: "c1",
			PreHook:     "echo pre1",
			PostHook:    "echo post1",
			Paths:       []string{"/path1"},
		},
		{
			ContainerID: "c2",
			PreHook:     "echo pre2",
			PostHook:    "",
			Paths:       []string{"/path2"},
		},
	}

	var executedArgs []string
	execRestic := func(ctx context.Context, args []string) error {
		executedArgs = args
		return nil
	}

	err := RunHooksAndBackup(context.Background(), client, targets, []string{"--tag", "test"}, execRestic)
	assert.NoError(t, err)
	
	// Check hooks
	assert.Len(t, client.execs, 3)
	assert.Equal(t, execCall{"c1", []string{"sh", "-c", "echo pre1"}}, client.execs[0])
	assert.Equal(t, execCall{"c2", []string{"sh", "-c", "echo pre2"}}, client.execs[1])
	assert.Equal(t, execCall{"c1", []string{"sh", "-c", "echo post1"}}, client.execs[2])

	// Check restic execution
	assert.Equal(t, []string{"backup", "/path1", "/path2", "--tag", "test"}, executedArgs)
}

func TestRunHooksAndBackup_ResticFailure(t *testing.T) {
	client := &mockDockerClient{}
	targets := []*apptypes.Target{
		{
			ContainerID: "c1",
			PreHook:     "echo pre1",
			PostHook:    "echo post1",
			Paths:       []string{"/path1"},
		},
	}

	execRestic := func(ctx context.Context, args []string) error {
		return fmt.Errorf("restic failed")
	}

	err := RunHooksAndBackup(context.Background(), client, targets, []string{}, execRestic)
	assert.ErrorContains(t, err, "restic backup failed: restic failed")
	
	// Check hooks - post hook should still run!
	assert.Len(t, client.execs, 2)
	assert.Equal(t, execCall{"c1", []string{"sh", "-c", "echo pre1"}}, client.execs[0])
	assert.Equal(t, execCall{"c1", []string{"sh", "-c", "echo post1"}}, client.execs[1])
}
