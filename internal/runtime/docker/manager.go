package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Manager struct {
	cli *client.Client
}

func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Manager{cli: cli}, nil
}

type StartRequest struct {
	ContainerName string
	Image         string
	PGDataPath    string
	HostPort      int
	User          string
	Password      string
	Database      string
}

func (m *Manager) Start(ctx context.Context, req StartRequest) (string, error) {
	// 1. Ensure image exists
	_, _, err := m.cli.ImageInspectWithRaw(ctx, req.Image)
	if err != nil {
		if client.IsErrNotFound(err) {
			// Pull image
			reader, err := m.cli.ImagePull(ctx, req.Image, image.PullOptions{})
			if err != nil {
				return "", fmt.Errorf("failed to pull image %s: %w", req.Image, err)
			}
			defer reader.Close()
			_, _ = io.Copy(io.Discard, reader) // wait for pull
		} else {
			return "", err
		}
	}

	// 2. Create container
	portString := fmt.Sprintf("%d/tcp", 5432)
	hostConfig := &container.HostConfig{
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
		PortBindings: nat.PortMap{
			nat.Port(portString): []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: fmt.Sprintf("%d", req.HostPort),
				},
			},
		},
		Binds: []string{
			fmt.Sprintf("%s:/var/lib/postgresql/data", req.PGDataPath),
		},
	}

	env := []string{
		fmt.Sprintf("POSTGRES_USER=%s", req.User),
		fmt.Sprintf("POSTGRES_PASSWORD=%s", req.Password),
		fmt.Sprintf("POSTGRES_DB=%s", req.Database),
	}

	cfg := &container.Config{
		Image: req.Image,
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(portString): struct{}{},
		},
		Labels: map[string]string{
			"pgv-managed": "true",
		},
	}

	resp, err := m.cli.ContainerCreate(ctx, cfg, hostConfig, &network.NetworkingConfig{}, &v1.Platform{}, req.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// 3. Start container
	if err := m.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

func (m *Manager) Stop(ctx context.Context, containerID string) error {
	timeout := 10 // seconds
	return m.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (m *Manager) Remove(ctx context.Context, containerID string) error {
	return m.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

func (m *Manager) Status(ctx context.Context, containerID string) (string, error) {
	inspect, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "not-found", nil
		}
		return "unknown", err
	}
	return inspect.State.Status, nil
}
