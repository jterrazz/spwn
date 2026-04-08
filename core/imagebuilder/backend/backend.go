package backend

import (
	"context"
	"io"
	"time"
)

// LogsConfig controls log streaming behavior.
type LogsConfig struct {
	Follow bool
	Tail   string
}

// ContainerConfig defines how to create a container.
type ContainerConfig struct {
	Image       string
	Name        string
	CPU         int64
	Memory      int64
	PidsLimit   int64
	NetworkMode string
	Binds       []string
	Env         []string
	ExtraHosts  []string // e.g. "host.docker.internal:host-gateway"
}

// ExecConfig defines a command to run inside a container.
type ExecConfig struct {
	Cmd []string
	Env []string
	TTY bool
}

// ImageInfo describes a Docker image.
type ImageInfo struct {
	Tag     string
	Size    int64
	Created time.Time
}

// ContainerInfo describes a running container's state.
type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	Status    string
	Running   bool
	StartedAt time.Time
}

// Backend abstracts the container runtime.
type Backend interface {
	Create(ctx context.Context, cfg ContainerConfig) (string, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	Exec(ctx context.Context, containerID string, cfg ExecConfig) (int, error)
	ExecOutput(ctx context.Context, containerID string, cmd []string) (string, error)
	CopyTo(ctx context.Context, containerID string, destPath string, content []byte) error
	IsRunning(ctx context.Context, containerID string) (bool, error)
	ImageExists(ctx context.Context, image string) (bool, error)
	EnsureImage(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, logw io.Writer) error
	EnsureImageWithContext(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, extraFiles map[string][]byte, logw io.Writer) error
	ImageVersion(ctx context.Context, image string, label string) (string, error)
	Logs(ctx context.Context, containerID string, cfg LogsConfig) (io.ReadCloser, error)
	ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error

	// Commit creates a snapshot image from a running container.
	Commit(ctx context.Context, containerID string, imageTag string) error

	// ImageList returns images matching a filter prefix.
	ImageList(ctx context.Context, prefix string) ([]ImageInfo, error)

	// ImageRemove removes a Docker image.
	ImageRemove(ctx context.Context, imageTag string) error

	// Inspect returns information about a container by name or ID.
	Inspect(ctx context.Context, nameOrID string) (*ContainerInfo, error)
}
