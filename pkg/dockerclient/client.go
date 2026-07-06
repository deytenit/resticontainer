package dockerclient

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerClient interface {
	ListContainers(ctx context.Context) ([]types.Container, error)
	InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error)
	ExecCommand(ctx context.Context, containerID string, cmd []string) error
	StopContainer(ctx context.Context, id string) error
	StartContainer(ctx context.Context, id string) error
}

type DefaultClient struct {
	cli *client.Client
}

func NewClient() (*DefaultClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DefaultClient{cli: cli}, nil
}

func (c *DefaultClient) ListContainers(ctx context.Context) ([]types.Container, error) {
	return c.cli.ContainerList(ctx, container.ListOptions{})
}

func (c *DefaultClient) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	return c.cli.ContainerInspect(ctx, id)
}

func (c *DefaultClient) ExecCommand(ctx context.Context, containerID string, cmd []string) error {
	execOpts := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}
	resp, err := c.cli.ContainerExecCreate(ctx, containerID, execOpts)
	if err != nil {
		return err
	}
	attach, err := c.cli.ContainerExecAttach(ctx, resp.ID, container.ExecStartOptions{})
	if err != nil {
		return err
	}
	defer attach.Close()
	_, _ = io.Copy(io.Discard, attach.Reader)
	return nil
}

func (c *DefaultClient) StopContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStop(ctx, id, container.StopOptions{})
}

func (c *DefaultClient) StartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}
