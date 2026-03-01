package backend

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/term"
)

// Docker implements Backend using the Docker Engine API.
type Docker struct {
	cli *client.Client
}

// NewDocker creates a new Docker backend.
func NewDocker() (*Docker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to Docker: %w", err)
	}
	return &Docker{cli: cli}, nil
}

func (d *Docker) Create(ctx context.Context, cfg ContainerConfig) (string, error) {
	// Resource limits.
	resources := container.Resources{
		NanoCPUs:  cfg.CPU * 1e9,
		Memory:    cfg.Memory,
		PidsLimit: &cfg.PidsLimit,
	}

	hostConfig := &container.HostConfig{
		Resources:   resources,
		NetworkMode: container.NetworkMode(cfg.NetworkMode),
		Binds:       cfg.Binds,
	}

	containerConfig := &container.Config{
		Image:      cfg.Image,
		Entrypoint: []string{"sleep", "infinity"},
		Env:        cfg.Env,
	}

	resp, err := d.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}

	return resp.ID, nil
}

func (d *Docker) Start(ctx context.Context, containerID string) error {
	return d.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (d *Docker) Stop(ctx context.Context, containerID string) error {
	return d.cli.ContainerStop(ctx, containerID, container.StopOptions{})
}

func (d *Docker) Remove(ctx context.Context, containerID string) error {
	return d.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

func (d *Docker) Exec(ctx context.Context, containerID string, cfg ExecConfig) (int, error) {
	execConfig := container.ExecOptions{
		Cmd:          cfg.Cmd,
		Env:          cfg.Env,
		AttachStdin:  cfg.TTY,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          cfg.TTY,
	}

	execResp, err := d.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return -1, fmt.Errorf("creating exec: %w", err)
	}

	attachResp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{Tty: cfg.TTY})
	if err != nil {
		return -1, fmt.Errorf("attaching to exec: %w", err)
	}
	defer attachResp.Close()

	if cfg.TTY {
		// Set terminal to raw mode for interactive use.
		fd := int(os.Stdin.Fd())
		oldState, err := term.MakeRaw(fd)
		if err == nil {
			defer term.Restore(fd, oldState)
		}

		go func() {
			io.Copy(attachResp.Conn, os.Stdin)
		}()
		io.Copy(os.Stdout, attachResp.Reader)
	} else {
		stdcopy.StdCopy(io.Discard, io.Discard, attachResp.Reader)
	}

	inspectResp, err := d.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return -1, fmt.Errorf("inspecting exec: %w", err)
	}

	return inspectResp.ExitCode, nil
}

func (d *Docker) CopyTo(ctx context.Context, containerID string, destPath string, content io.Reader) error {
	// Read all content.
	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("reading content: %w", err)
	}

	// Create a tar archive with the file.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "physics.md",
		Mode: 0o644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing tar header: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("writing tar content: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("closing tar: %w", err)
	}

	return d.cli.CopyToContainer(ctx, containerID, destPath, &buf, types.CopyToContainerOptions{})
}

func (d *Docker) IsRunning(ctx context.Context, containerID string) (bool, error) {
	info, err := d.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	return info.State.Running, nil
}

func (d *Docker) ImageExists(ctx context.Context, img string) (bool, error) {
	_, _, err := d.cli.ImageInspectWithRaw(ctx, img)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *Docker) ExecOutput(ctx context.Context, containerID string, cmd []string) (string, error) {
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: false,
	}

	execResp, err := d.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("creating exec: %w", err)
	}

	attachResp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("attaching to exec: %w", err)
	}
	defer attachResp.Close()

	var stdout bytes.Buffer
	stdcopy.StdCopy(&stdout, io.Discard, attachResp.Reader)
	return strings.TrimSpace(stdout.String()), nil
}

func (d *Docker) Logs(ctx context.Context, containerID string, cfg LogsConfig) (io.ReadCloser, error) {
	return d.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     cfg.Follow,
		Tail:       cfg.Tail,
	})
}

func (d *Docker) ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error {
	execConfig := container.ExecOptions{
		Cmd:    cfg.Cmd,
		Env:    cfg.Env,
		Detach: true,
	}

	execResp, err := d.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("creating detached exec: %w", err)
	}

	return d.cli.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{Detach: true})
}

// PullImage pulls an image from a registry.
func (d *Docker) PullImage(ctx context.Context, ref string) error {
	reader, err := d.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %s: %w", ref, err)
	}
	defer reader.Close()
	// Drain the reader to complete the pull.
	io.Copy(io.Discard, reader)
	return nil
}
