package apfs

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/src/platform/local"
)

const (
	pgrepWait = 2 * time.Second
	// value: the APFS volume of the owner's data; / is the sealed system's.
	dataVolume = "/System/Volumes/Data"
	// value: the disk of a staged system root, a Mac's of 1 TiB with half of
	// it free.
	stagedTotal = 1 << 40
	stagedFree  = 1 << 39
)

type Commands interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Usage struct {
	run Commands
	// value: the system root the Mac's own folders are read under, "" for the
	// real one.
	root string
}

func New(run Commands, root string) *Usage { return &Usage{run: run, root: root} }

func (u *Usage) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	local.Sizes(ctx, roots, each)
}

func (u *Usage) Volume(context.Context) (total, free int64, err error) {
	// invariant: a staged root stands on no volume of this host: its disk is
	// the one the stage says, so no read of it tells the free space of the
	// machine that happens to run it.
	if u.root != "" {
		return stagedTotal, stagedFree, nil
	}
	var st unix.Statfs_t
	path := dataVolume
	for {
		err = unix.Statfs(path, &st)
		if !errors.Is(err, fs.ErrNotExist) || path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	if err != nil {
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
