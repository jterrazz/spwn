package backend

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imageTypes "github.com/docker/docker/api/types/image"
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

// Create provisions a new container with the given configuration and returns its ID.
func (d *Docker) Create(ctx context.Context, cfg ContainerConfig) (string, error) {
	hostCfg := &containerTypes.HostConfig{
		Resources: containerTypes.Resources{
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
		Labels:     cfg.Labels,
	}

	resp, err := d.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	return resp.ID, nil
}

// Start starts a previously created container.
func (d *Docker) Start(ctx context.Context, containerID string) error {
	return d.client.ContainerStart(ctx, containerID, containerTypes.StartOptions{})
}

// Stop gracefully stops a running container.
func (d *Docker) Stop(ctx context.Context, containerID string) error {
	return d.client.ContainerStop(ctx, containerID, containerTypes.StopOptions{})
}

// Remove forcibly removes a container.
func (d *Docker) Remove(ctx context.Context, containerID string) error {
	return d.client.ContainerRemove(ctx, containerID, containerTypes.RemoveOptions{Force: true})
}

// Exec runs a command inside a container and returns the exit code.
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

// ExecOutput runs a command inside a container and returns its stdout as a string.
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

// CopyTo writes content into a file at destPath inside the container.
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

func (d *Docker) ImageVersion(ctx context.Context, image string, label string) (string, error) {
	inspect, _, err := d.client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if inspect.Config == nil || inspect.Config.Labels == nil {
		return "", nil
	}
	return inspect.Config.Labels[label], nil
}

func (d *Docker) EnsureImage(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, logw io.Writer) error {
	return d.EnsureImageWithContext(ctx, tag, expectedVersion, dockerfile, nil, logw)
}

func (d *Docker) EnsureImageWithContext(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, extraFiles map[string][]byte, logw io.Writer) error {
	if logw == nil {
		logw = io.Discard
	}

	// Check current version
	currentVersion, err := d.ImageVersion(ctx, tag, "sh.spwn.image-version")
	if err != nil {
		return fmt.Errorf("check image version: %w", err)
	}

	if !NeedsRebuild(currentVersion, expectedVersion) {
		return nil
	}

	// Log what we're doing
	if currentVersion == "" {
		fmt.Fprintf(logw, "Building %s (v%s)...\n", tag, expectedVersion)
	} else {
		fmt.Fprintf(logw, "Rebuilding %s (v%s → v%s)...\n", tag, currentVersion, expectedVersion)
		// Remove old image before rebuilding
		_ = d.ImageRemove(ctx, tag)
	}

	// Create tar context containing the Dockerfile and any extra files
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	hdr := &tar.Header{Name: "Dockerfile", Size: int64(len(dockerfile)), Mode: 0644}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header: %w", err)
	}
	if _, err := tw.Write(dockerfile); err != nil {
		return fmt.Errorf("tar write: %w", err)
	}

	// Add extra context files (create parent directories first)
	dirs := make(map[string]bool)
	for name := range extraFiles {
		parts := strings.Split(filepath.Dir(name), string(filepath.Separator))
		for i := range parts {
			d := strings.Join(parts[:i+1], "/")
			if d != "." && !dirs[d] {
				dirs[d] = true
				dirHdr := &tar.Header{Name: d + "/", Typeflag: tar.TypeDir, Mode: 0755}
				tw.WriteHeader(dirHdr)
			}
		}
	}
	for name, content := range extraFiles {
		fileHdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0755}
		if err := tw.WriteHeader(fileHdr); err != nil {
			return fmt.Errorf("tar header %s: %w", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			return fmt.Errorf("tar write %s: %w", name, err)
		}
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

	// Stream build output - Docker returns JSON lines with a "stream" field.
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

func (d *Docker) Commit(ctx context.Context, containerID string, imageTag string) error {
	_, err := d.client.ContainerCommit(ctx, containerID, containerTypes.CommitOptions{
		Reference: imageTag,
		Comment:   "spwn world snapshot",
	})
	return err
}

func (d *Docker) ImageList(ctx context.Context, prefix string) ([]ImageInfo, error) {
	images, err := d.client.ImageList(ctx, imageTypes.ListOptions{})
	if err != nil {
		return nil, err
	}
	var result []ImageInfo
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if strings.HasPrefix(tag, prefix) {
				result = append(result, ImageInfo{
					Tag:     tag,
					Size:    img.Size,
					Created: time.Unix(img.Created, 0),
				})
			}
		}
	}
	return result, nil
}

func (d *Docker) ImageRemove(ctx context.Context, imageTag string) error {
	_, err := d.client.ImageRemove(ctx, imageTag, imageTypes.RemoveOptions{Force: true})
	return err
}

// Inspect returns information about a container by name or ID.
func (d *Docker) Inspect(ctx context.Context, nameOrID string) (*ContainerInfo, error) {
	info, err := d.client.ContainerInspect(ctx, nameOrID)
	if err != nil {
		return nil, err
	}

	var startedAt time.Time
	if info.State != nil && info.State.StartedAt != "" {
		startedAt, _ = time.Parse(time.RFC3339Nano, info.State.StartedAt)
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, info.Created)

	name := info.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	status := ""
	running := false
	if info.State != nil {
		status = info.State.Status
		running = info.State.Running
	}

	var labels map[string]string
	if info.Config != nil {
		labels = info.Config.Labels
	}

	return &ContainerInfo{
		ID:        info.ID,
		Name:      name,
		Image:     info.Config.Image,
		Status:    status,
		Running:   running,
		StartedAt: startedAt,
		CreatedAt: createdAt,
		Labels:    labels,
	}, nil
}

// ListContainersByLabel enumerates every container (running OR stopped)
// whose Docker labels include key=value. Used by the state store to
// list spwn worlds straight from the daemon - no JSON file involved,
// no possibility of drift after `docker rm`.
func (d *Docker) ListContainersByLabel(ctx context.Context, key, value string) ([]ContainerInfo, error) {
	args := filters.NewArgs()
	if value == "" {
		args.Add("label", key)
	} else {
		args.Add("label", key+"="+value)
	}

	containers, err := d.client.ContainerList(ctx, containerTypes.ListOptions{
		All:     true, // include stopped - a stopped world is still a world
		Filters: args,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	out := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		out = append(out, ContainerInfo{
			ID:        c.ID,
			Name:      name,
			Image:     c.Image,
			Status:    c.Status,
			Running:   c.State == "running",
			CreatedAt: time.Unix(c.Created, 0),
			Labels:    c.Labels,
		})
	}
	return out, nil
}

// CreateNamedContainer creates a container with explicit config and host config.
// Unlike Create(), this allows full control (e.g., custom restart policy, entrypoint).
func (d *Docker) CreateNamedContainer(ctx context.Context, name string, cfg *containerTypes.Config, hostCfg *containerTypes.HostConfig) (string, error) {
	resp, err := d.client.ContainerCreate(ctx, cfg, hostCfg, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	return resp.ID, nil
}
