package backend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/mind"
)

// DockerBackend implements Backend using the Docker Engine API.
type DockerBackend struct {
	client *client.Client
}

// NewDockerBackend creates a new Docker backend.
func NewDockerBackend() (*DockerBackend, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to docker: %w", err)
	}
	return &DockerBackend{client: cli}, nil
}

func (d *DockerBackend) Create(ctx context.Context, cfg *config.UniverseConfig) (string, error) {
	// Pull image if not available locally.
	if err := d.ensureImage(ctx, cfg.Image); err != nil {
		return "", err
	}

	id := uuid.New().String()
	shortID := id[:8]
	containerName := "universe-" + shortID

	labels := map[string]string{
		"universe.id": id,
	}

	var mounts []mount.Mount

	if cfg.Mind != "" {
		labels["universe.mind"] = cfg.Mind
		mindPath, err := mind.ResolvePath(cfg.Mind)
		if err != nil {
			return "", err
		}
		if err := mind.EnsureDir(mindPath); err != nil {
			return "", err
		}
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: mindPath,
			Target: "/mind",
		})
	}

	if cfg.Workspace != "" {
		labels["universe.workspace"] = cfg.Workspace
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: cfg.Workspace,
			Target: "/workspace",
		})
	}

	containerCfg := &container.Config{
		Image:     cfg.Image,
		Labels:    labels,
		Tty:       true,
		OpenStdin: true,
		Cmd:       []string{"sleep", "infinity"},
	}

	hostCfg := &container.HostConfig{
		Mounts: mounts,
	}

	resp, err := d.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}
	_ = resp // container ID is Docker's; we use our own UUID

	return id, nil
}

func (d *DockerBackend) Start(ctx context.Context, id string) error {
	name, err := d.containerName(ctx, id)
	if err != nil {
		return err
	}
	return d.client.ContainerStart(ctx, name, container.StartOptions{})
}

func (d *DockerBackend) Exec(ctx context.Context, id string, cmd []string) (*ExecResult, error) {
	name, err := d.containerName(ctx, id)
	if err != nil {
		return nil, err
	}

	execCfg := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := d.client.ContainerExecCreate(ctx, name, execCfg)
	if err != nil {
		return nil, fmt.Errorf("creating exec: %w", err)
	}

	attachResp, err := d.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("attaching to exec: %w", err)
	}
	defer attachResp.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, attachResp.Reader)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading exec output: %w", err)
	}

	inspectResp, err := d.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("inspecting exec: %w", err)
	}

	return &ExecResult{
		ExitCode: inspectResp.ExitCode,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
	}, nil
}

func (d *DockerBackend) Stop(ctx context.Context, id string) error {
	name, err := d.containerName(ctx, id)
	if err != nil {
		return err
	}
	timeout := 10
	return d.client.ContainerStop(ctx, name, container.StopOptions{Timeout: &timeout})
}

func (d *DockerBackend) Remove(ctx context.Context, id string) error {
	name, err := d.containerName(ctx, id)
	if err != nil {
		return err
	}
	return d.client.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
}

func (d *DockerBackend) List(ctx context.Context) ([]ContainerInfo, error) {
	f := filters.NewArgs()
	f.Add("label", "universe.id")

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	var infos []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		infos = append(infos, ContainerInfo{
			ID:        c.Labels["universe.id"],
			Name:      name,
			Image:     c.Image,
			Status:    c.State,
			Mind:      c.Labels["universe.mind"],
			Workspace: c.Labels["universe.workspace"],
		})
	}

	return infos, nil
}

func (d *DockerBackend) Inspect(ctx context.Context, id string) (*ContainerInfo, error) {
	name, err := d.containerName(ctx, id)
	if err != nil {
		return nil, err
	}

	detail, err := d.client.ContainerInspect(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("inspecting container: %w", err)
	}

	labels := detail.Config.Labels
	status := "unknown"
	if detail.State != nil {
		status = detail.State.Status
	}

	return &ContainerInfo{
		ID:        labels["universe.id"],
		Name:      strings.TrimPrefix(detail.Name, "/"),
		Image:     detail.Config.Image,
		Status:    status,
		Mind:      labels["universe.mind"],
		Workspace: labels["universe.workspace"],
	}, nil
}

func (d *DockerBackend) ensureImage(ctx context.Context, image string) error {
	_, _, err := d.client.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil // image exists locally
	}

	reader, err := d.client.ImagePull(ctx, image, imagetypes.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %s: %w", image, err)
	}
	defer reader.Close()
	// Drain the reader to complete the pull.
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func (d *DockerBackend) containerName(ctx context.Context, universeID string) (string, error) {
	containers, err := d.List(ctx)
	if err != nil {
		return "", err
	}
	for _, c := range containers {
		if c.ID == universeID {
			return c.Name, nil
		}
	}
	return "", fmt.Errorf("universe not found: %s", universeID)
}
