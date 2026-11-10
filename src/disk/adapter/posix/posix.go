package posix

import (
	"context"
	"time"

	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/src/platform/local"
)

const (
	pgrepWait = 2 * time.Second
	// value: a Linux machine's one volume.
	dataVolume = "/"
)

type Commands interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Usage struct{ run Commands }

func New(run Commands) *Usage { return &Usage{run: run} }

func (u *Usage) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	local.Sizes(ctx, roots, each)
}

func (u *Usage) Volume(context.Context) (total, free int64, err error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dataVolume, &st); err != nil {
		return 0, 0, err
	}
	block := int64(st.Bsize)                                       //nolint:unconvert // Bsize is a uint32 on darwin, an int64 on linux.
	return int64(st.Blocks) * block, int64(st.Bavail) * block, nil //nolint:gosec // outside: a volume's block counts fit an int64.
}

func (u *Usage) Move(_ context.Context, from, to string) error { return local.Move(from, to) }

func (u *Usage) Erase(_ context.Context, paths []string) error { return local.Erase(paths) }

func (u *Usage) Running(ctx context.Context, process string) bool {
	out, err := u.run.Output(ctx, local.Cmd{Name: "pgrep", Args: []string{"-x", process}, Timeout: pgrepWait})
	return err == nil && out.ExitCode == 0
}
