package backend

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/term"
)

// Docker implements Backend using the Docker Engine API.
type Docker struct {
	client *client.Client
}

// NewDocker creates a Docker backend from the environment.
func NewDocker() (*Docker, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	_, err = c.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("docker not reachable: %w", err)
	}
	return &Docker{client: c}, nil
}

func (d *Docker) Create(ctx context.Context, cfg ContainerConfig) (string, error) {
	hostCfg := &containerTypes.HostConfig{
		Resources: containerTypes.Resources{
			NanoCPUs:  cfg.CPU * 1e9,
			Memory:    cfg.Memory,
			PidsLimit: &cfg.PidsLimit,
		},
		NetworkMode: containerTypes.NetworkMode(cfg.NetworkMode),
		Binds:       cfg.Binds,
		ExtraHosts:  cfg.ExtraHosts,
	}

	containerCfg := &containerTypes.Config{
		Image:      cfg.Image,
		Entrypoint: []string{"sleep", "infinity"},
		Env:        cfg.Env,
	}

	resp, err := d.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	return resp.ID, nil
}

func (d *Docker) Start(ctx context.Context, containerID string) error {
	return d.client.ContainerStart(ctx, containerID, containerTypes.StartOptions{})
}

func (d *Docker) Stop(ctx context.Context, containerID string) error {
	return d.client.ContainerStop(ctx, containerID, containerTypes.StopOptions{})
}

func (d *Docker) Remove(ctx context.Context, containerID string) error {
	return d.client.ContainerRemove(ctx, containerID, containerTypes.RemoveOptions{Force: true})
}

func (d *Docker) Exec(ctx context.Context, containerID string, cfg ExecConfig) (int, error) {
	execCfg := types.ExecConfig{
		Cmd:          cfg.Cmd,
		Env:          cfg.Env,
		Tty:          cfg.TTY,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := d.client.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return -1, fmt.Errorf("exec create: %w", err)
	}

	resp, err := d.client.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{Tty: cfg.TTY})
	if err != nil {
		return -1, fmt.Errorf("exec attach: %w", err)
	}
	defer resp.Close()

	if cfg.TTY {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer term.Restore(int(os.Stdin.Fd()), oldState)
		}
		go io.Copy(resp.Conn, os.Stdin)
		io.Copy(os.Stdout, resp.Reader)
	} else {
		go io.Copy(resp.Conn, os.Stdin)
		stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader)
	}

	inspect, err := d.client.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return -1, fmt.Errorf("exec inspect: %w", err)
	}
	return inspect.ExitCode, nil
}

func (d *Docker) ExecOutput(ctx context.Context, containerID string, cmd []string) (string, error) {
	execCfg := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := d.client.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return "", fmt.Errorf("exec create: %w", err)
	}

	resp, err := d.client.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("exec attach: %w", err)
	}
	defer resp.Close()

	var buf bytes.Buffer
	stdcopy.StdCopy(&buf, io.Discard, resp.Reader)
	output := strings.TrimSpace(buf.String())

	// Check exit code
	inspect, err := d.client.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return output, fmt.Errorf("exec inspect: %w", err)
	}
	if inspect.ExitCode != 0 {
		return output, fmt.Errorf("exit code %d", inspect.ExitCode)
	}

	return output, nil
}

func (d *Docker) CopyTo(ctx context.Context, containerID string, destPath string, content []byte) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: destPath,
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	tw.Close()

	return d.client.CopyToContainer(ctx, containerID, "/", &buf, types.CopyToContainerOptions{})
}

func (d *Docker) IsRunning(ctx context.Context, containerID string) (bool, error) {
	info, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	return info.State.Running, nil
}

func (d *Docker) ImageExists(ctx context.Context, image string) (bool, error) {
	_, _, err := d.client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *Docker) EnsureImage(ctx context.Context, tag string, dockerfile []byte, logw io.Writer) error {
	exists, err := d.ImageExists(ctx, tag)
	if err != nil {
		return fmt.Errorf("check image: %w", err)
	}
	if exists {
		return nil
	}

	if logw == nil {
		logw = io.Discard
	}

	// Create tar context containing only the Dockerfile
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	hdr := &tar.Header{Name: "Dockerfile", Size: int64(len(dockerfile)), Mode: 0644}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header: %w", err)
	}
	if _, err := tw.Write(dockerfile); err != nil {
		return fmt.Errorf("tar write: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar close: %w", err)
	}

	resp, err := d.client.ImageBuild(ctx, buf, types.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("build image: %w", err)
	}
	defer resp.Body.Close()

	// Stream build output — Docker returns JSON lines with a "stream" field.
	type buildMsg struct {
		Stream string `json:"stream"`
		Error  string `json:"error"`
	}
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg buildMsg
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			return fmt.Errorf("image build: %s", msg.Error)
		}
		if msg.Stream != "" {
			fmt.Fprint(logw, msg.Stream)
		}
	}

	return nil
}

func (d *Docker) Logs(ctx context.Context, containerID string, cfg LogsConfig) (io.ReadCloser, error) {
	opts := containerTypes.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     cfg.Follow,
		Tail:       cfg.Tail,
	}
	return d.client.ContainerLogs(ctx, containerID, opts)
}

func (d *Docker) ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error {
	execCfg := types.ExecConfig{
		Cmd:          cfg.Cmd,
		Env:          cfg.Env,
		Tty:          cfg.TTY,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Detach:       true,
	}

	execID, err := d.client.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return fmt.Errorf("exec create: %w", err)
	}

	return d.client.ContainerExecStart(ctx, execID.ID, types.ExecStartCheck{Detach: true})
}
