package backend

import (
	"context"

	"github.com/jterrazz/universe/internal/config"
)

// ContainerInfo holds metadata about a container.
type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	Status    string
	Mind      string
	Workspace string
}

// ExecResult holds the output of a command execution.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Backend defines the interface for container runtimes.
type Backend interface {
	Create(ctx context.Context, cfg *config.UniverseConfig) (string, error)
	Start(ctx context.Context, id string) error
	Exec(ctx context.Context, id string, cmd []string) (*ExecResult, error)
	Stop(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	List(ctx context.Context) ([]ContainerInfo, error)
	Inspect(ctx context.Context, id string) (*ContainerInfo, error)
}
