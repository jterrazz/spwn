package backend

import (
	"context"
	"io"
)

// LogsConfig holds parameters for streaming container logs.
type LogsConfig struct {
	Follow bool
	Tail   string // number of lines, e.g. "100" or "all"
}

// ContainerConfig holds the parameters for creating a container.
type ContainerConfig struct {
	Image        string
	Name         string
	CPU          int64             // number of cores
	Memory       int64             // bytes
	PidsLimit    int64
	NetworkMode  string            // "none", "bridge", "host"
	Binds        []string          // host:container mount specs
	Env          []string          // environment variables
}

// ExecConfig holds the parameters for executing a command in a container.
type ExecConfig struct {
	Cmd  []string
	Env  []string
	TTY  bool
}

// Backend defines the interface for container runtimes.
type Backend interface {
	// Create provisions a new container and returns its ID.
	Create(ctx context.Context, cfg ContainerConfig) (string, error)

	// Start starts a stopped container.
	Start(ctx context.Context, containerID string) error

	// Stop stops a running container.
	Stop(ctx context.Context, containerID string) error

	// Remove removes a container.
	Remove(ctx context.Context, containerID string) error

	// Exec runs a command inside a container interactively (with TTY).
	Exec(ctx context.Context, containerID string, cfg ExecConfig) (int, error)

	// CopyTo copies content into a container at the given path.
	CopyTo(ctx context.Context, containerID string, destPath string, content io.Reader) error

	// IsRunning checks if a container is running.
	IsRunning(ctx context.Context, containerID string) (bool, error)

	// ImageExists checks if a Docker image exists locally.
	ImageExists(ctx context.Context, image string) (bool, error)

	// ExecOutput runs a command inside a container and returns its stdout.
	ExecOutput(ctx context.Context, containerID string, cmd []string) (string, error)

	// Logs streams container stdout/stderr.
	Logs(ctx context.Context, containerID string, cfg LogsConfig) (io.ReadCloser, error)

	// ExecDetached starts a command inside a container without waiting for completion.
	ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error
}
