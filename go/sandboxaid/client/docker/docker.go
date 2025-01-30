package docker

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	sclient "github.com/substratusai/sandboxai/sandboxaid/client"

	stdlog "log"

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "github.com/substratusai/sandboxai/api/v1"
)

var _ sclient.Client = &DockerClient{}

type Logger interface {
	Printf(format string, v ...interface{})
}

func SetLogger(logger Logger) {
	log = logger
}

var log Logger = stdlog.New(os.Stderr, "", stdlog.LstdFlags)

type DockerClient struct {
	docker *dclient.Client
	httpc  *http.Client
	scope  string
}

func NewSandboxClient(docker *dclient.Client, httpc *http.Client, scope string) (*DockerClient, error) {
	if docker == nil {
		if err := setDockerEnvFromContextIfNotSet(); err != nil {
			log.Printf("Failed to set docker env from context: %v", err)
		}
		var err error
		docker, err = dclient.NewClientWithOpts(dclient.FromEnv)
		if err != nil {
			return nil, err
		}
	}

	return &DockerClient{
		docker: docker,
		httpc:  httpc,
		scope:  scope,
	}, nil
}

const labelKeyScope = "sandboxai.scope"
const labelKeySpace = "sandboxai.space"
const labelKeyName = "sandboxai.name"

func (c *DockerClient) CreateSandbox(ctx context.Context, space string, req *v1.CreateSandboxRequest) (*sclient.Sandbox, error) {
	if space == "" {
		return nil, fmt.Errorf("space cannot be empty")
	}
	if req.Name == "" {
		req.Name = generateRandomName()
	}
	cname := containerName(space, req.Name)

	const boxPortNumber = "8000"
	boxPort, err := nat.NewPort("tcp", boxPortNumber)
	if err != nil {
		return nil, fmt.Errorf("create port: %w", err)
	}

	var env []string
	for k, v := range req.Spec.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	config := &container.Config{
		Image: req.Spec.Image,
		ExposedPorts: nat.PortSet{
			boxPort: struct{}{},
		},
		Labels: map[string]string{
			labelKeyScope: c.scope,
			labelKeySpace: space,
			labelKeyName:  req.Name,
		},
		Env: env,
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			boxPortNumber: []nat.PortBinding{
				{
					HostIP: "127.0.0.1",
					// Find a free port on the host machine.
					HostPort: "0",
				},
			},
		},
		PublishAllPorts: true,
	}

	networkingConfig := &network.NetworkingConfig{}
	platform := &ocispec.Platform{}

	resp, err := c.docker.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, cname)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	startOpts := container.StartOptions{}
	if err := c.docker.ContainerStart(ctx, resp.ID, startOpts); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	log.Printf("Started sandbox: %q", resp.ID)

	dockerContainer, err := c.docker.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, err
	}
	created, err := containerJSONToSandbox(dockerContainer)
	if err != nil {
		return nil, fmt.Errorf("reading container to sandbox: %w", err)
	}
	if err := c.waitForHealthcheck(ctx, created.BoxHostPort, 1*time.Second, 60*time.Second); err != nil {
		return nil, fmt.Errorf("waiting for box healthcheck: %w", err)
	}

	log.Printf("Sandbox ready: %q", resp.ID)

	return created, nil
}

func (c *DockerClient) GetSandbox(ctx context.Context, space, name string) (*sclient.Sandbox, error) {
	if space == "" {
		return nil, fmt.Errorf("space cannot be empty")
	}
	cname := containerName(space, name)
	dockerContainer, err := c.docker.ContainerInspect(ctx, cname)
	if err != nil {
		if dclient.IsErrNotFound(err) {
			return nil, fmt.Errorf("getting container %q: %w", cname, sclient.ErrSandboxNotFound)
		}
		return nil, fmt.Errorf("getting container %q: %w", cname, err)
	}
	return containerJSONToSandbox(dockerContainer)
}

func (c *DockerClient) DeleteSandbox(ctx context.Context, space, name string) error {
	if space == "" {
		return fmt.Errorf("space cannot be empty")
	}
	cname := containerName(space, name)
	if err := c.docker.ContainerStop(ctx, cname, container.StopOptions{
		// TODO: Configurable timeout.
		// Timeout:
	}); err != nil {
		if dclient.IsErrNotFound(err) {
			return fmt.Errorf("getting container %q: %w", cname, sclient.ErrSandboxNotFound)
		}
		return fmt.Errorf("stoping container %q: %w", cname, err)
	}
	if err := c.docker.ContainerRemove(ctx, containerName(space, name), container.RemoveOptions{}); err != nil {
		if dclient.IsErrNotFound(err) {
			return fmt.Errorf("removing container %q: %w", cname, sclient.ErrSandboxNotFound)
		}
		return fmt.Errorf("removing container %q: %w", cname, err)
	}
	return nil
}

type SandboxSpacedName struct {
	Space string
	Name  string
}

func (c *DockerClient) ListAllSandboxes(ctx context.Context) ([]SandboxSpacedName, error) {
	containers, err := c.docker.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("%s=%s", labelKeyScope, c.scope)),
		),
	})
	if err != nil {
		return nil, err
	}

	items := make([]SandboxSpacedName, len(containers))
	for i, container := range containers {
		items[i] = SandboxSpacedName{
			Space: container.Labels[labelKeySpace],
			Name:  container.Names[0],
		}
	}
	return items, nil
}

func (c *DockerClient) waitForHealthcheck(ctx context.Context, port int, interval, timeout time.Duration) error {
	start := time.Now()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("healthcheck timeout")
		}
		if err := c.sendHealthcheck(ctx, port); err == nil {
			return nil
		}
	}
}

func (c *DockerClient) sendHealthcheck(ctx context.Context, port int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/healthz", port), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck failed: %d", resp.StatusCode)
	}
	return nil
}
