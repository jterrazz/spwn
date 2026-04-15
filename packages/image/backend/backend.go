package backend

import (
	"context"
	"io"
	"time"
)

// ContainerConfig defines how to create a container. CPU and memory
// limits are intentionally absent: worlds inherit the Docker host
// defaults. A per-world hard-limit knob may return in the future, but
// it does not live in spwn.yaml today.
type ContainerConfig struct {
	Image       string
	Name        string
	PidsLimit   int64
	NetworkMode string
	Binds       []string
	Env         []string
	ExtraHosts  []string // e.g. "host.docker.internal:host-gateway"
	// Labels is the set of Docker labels written to the container at
	// create time. spwn uses labels as the canonical store for world
	// metadata so the daemon itself becomes the source of truth.
	Labels map[string]string
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

// ContainerInfo describes a container's state. Labels is populated by
// list/inspect calls so callers can identify spwn-tagged containers
// without a second round-trip.
type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	Status    string
	Running   bool
	StartedAt time.Time
	CreatedAt time.Time
	Labels    map[string]string
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

	// CopyDirTo copies the contents of a host directory into a
	// directory inside the container. The destination directory is
	// created if it doesn't exist. Files that already exist at the
	// destination are overwritten. This is the mechanism used to
	// seed per-agent home directories from `spwn/agents/<name>/` on
	// the host into `/agents/<name>/` inside the container at spawn
	// time — the committed agent tree flows in via a one-time copy,
	// not via a bind mount.
	CopyDirTo(ctx context.Context, containerID string, destDir string, hostSrcDir string) error

	// CopyDirFrom copies the contents of a directory inside the
	// container back to a host directory. Used by `spwn down` to
	// snapshot an agent's durable memory layers (journal, knowledge,
	// playbooks, skills) out of the container before it's destroyed.
	// hostDestDir is created if it doesn't exist; existing files at
	// the destination are overwritten.
	CopyDirFrom(ctx context.Context, containerID string, srcDir string, hostDestDir string) error

	IsRunning(ctx context.Context, containerID string) (bool, error)
	ImageExists(ctx context.Context, image string) (bool, error)
	// EnsureImage and EnsureImageWithContext return (rebuilt, err).
	// rebuilt=true means the build actually ran; false means the
	// tagged image already matched expectedVersion and was reused.
	// Callers use this to emit accurate progress messages.
	EnsureImage(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, logw io.Writer) (bool, error)
	EnsureImageWithContext(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, extraFiles map[string][]byte, logw io.Writer) (bool, error)
	ImageVersion(ctx context.Context, image string, label string) (string, error)
	ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error

	// Commit creates a snapshot image from a running container.
	Commit(ctx context.Context, containerID string, imageTag string) error

	// ImageList returns images matching a filter prefix.
	ImageList(ctx context.Context, prefix string) ([]ImageInfo, error)

	// ImageRemove removes a Docker image.
	ImageRemove(ctx context.Context, imageTag string) error

	// Inspect returns information about a container by name or ID.
	Inspect(ctx context.Context, nameOrID string) (*ContainerInfo, error)

	// ListContainersByLabel returns every container (running or stopped)
	// whose Docker labels match the given key=value selector. spwn uses
	// this to enumerate worlds straight from the daemon, so the live
	// container set is the source of truth - no JSON state file to drift.
	ListContainersByLabel(ctx context.Context, key, value string) ([]ContainerInfo, error)
}
