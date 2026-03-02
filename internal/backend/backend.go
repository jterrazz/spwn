package backend

import (
	"context"
	"io"
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
	EnsureImage(ctx context.Context, tag string, dockerfile []byte, logw io.Writer) error
	Logs(ctx context.Context, containerID string, cfg LogsConfig) (io.ReadCloser, error)
	ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error
}
