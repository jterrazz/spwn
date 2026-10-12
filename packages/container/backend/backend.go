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
	ExtraHosts  []string
	// Labels is the set of Docker labels written to the container at
	// create time. spwn uses labels as the canonical store for world
	// metadata so the daemon itself becomes the source of truth.
	Labels map[string]string
	// GroupAdd are additional supplementary GIDs the container's main
	// user receives at startup. Used by chief-mode worlds to grant
	// the spwn user access to the bind-mounted /var/run/docker.sock
	// (which is owned by the host's root:docker group, GID = host's
	// docker group, NOT the container's "spwn" GID). Equivalent to
	// `docker run --group-add <gid>`.
	GroupAdd []string
}

// ExecConfig defines a command to run inside a container.
type ExecConfig struct {
	Cmd []string
	Env []string
	TTY bool
	// Stdout / Stderr, when non-nil, receive the runtime's output
	// instead of being copied to os.Stdout/os.Stderr. The
	// automation dispatcher uses this to capture the rendered
	// runtime's output into a receipt's `output` field for
	// dashboard display. Both streams are written separately so
	// callers can route them to different sinks (most callers
	// just want both into one bytes.Buffer).
	Stdout io.Writer
	Stderr io.Writer
}

// ImageInfo describes a Docker compile.
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

	CopyDirTo(ctx context.Context, containerID string, destDir string, hostSrcDir string) error

	CopyDirFrom(ctx context.Context, containerID string, srcDir string, hostDestDir string) error

	IsRunning(ctx context.Context, containerID string) (bool, error)
	ImageExists(ctx context.Context, image string) (bool, error)
	EnsureImage(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, logw io.Writer) (bool, error)
	EnsureImageWithContext(ctx context.Context, tag string, expectedVersion string, dockerfile []byte, extraFiles map[string][]byte, logw io.Writer) (bool, error)
	ImageVersion(ctx context.Context, image string, label string) (string, error)
	ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error

	Commit(ctx context.Context, containerID string, imageTag string) error

	ImageList(ctx context.Context, prefix string) ([]ImageInfo, error)

	ImageRemove(ctx context.Context, imageTag string) error

	Inspect(ctx context.Context, nameOrID string) (*ContainerInfo, error)

	ListContainersByLabel(ctx context.Context, key, value string) ([]ContainerInfo, error)
}
