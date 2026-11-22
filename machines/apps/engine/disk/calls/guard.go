package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/contract"
)

var errOtherAppsData = errors.New("inside another app's container, which macOS blocks Machines from reading")

// invariant: every read and change the survey makes on the machine passes
// here, and none reaches inside another app's container
// (catalog.InOtherAppsData): such a path globs to nothing, exists not, sizes
// 0, and is never moved nor erased.
type guardedMachine struct {
	Machine
	expand disk.Expander
}

type guardedUsage struct {
	contract.DiskUsage
	expand disk.Expander
}

func guarded(e disk.Expander, abs string) bool { return catalog.InOtherAppsData(e.Display(abs)) }

func (m guardedMachine) Glob(pattern string) []string {
	if guarded(m.expand, pattern) {
		return nil
	}
	return m.Machine.Glob(pattern)
}

func (m guardedMachine) Exists(p string) bool { return !guarded(m.expand, p) && m.Machine.Exists(p) }

func (m guardedMachine) Names(dir string) ([]string, error) {
	if guarded(m.expand, dir) {
		return nil, errOtherAppsData
	}
	return m.Machine.Names(dir)
}

func (m guardedMachine) ReadFile(p string) ([]byte, error) {
	if guarded(m.expand, p) {
		return nil, errOtherAppsData
	}
	return m.Machine.ReadFile(p)
}

func (m guardedMachine) Link(p string) (string, bool) {
	if guarded(m.expand, p) {
		return "", false
	}
	return m.Machine.Link(p)
}

func (m guardedMachine) Prune(dir string) {
	if !guarded(m.expand, dir) {
		m.Machine.Prune(dir)
	}
}

func (m guardedMachine) Keep(p string, data []byte) error {
	if guarded(m.expand, p) {
		return errOtherAppsData
	}
	return m.Machine.Keep(p, data)
}

func (u guardedUsage) Sizes(ctx context.Context, roots []string, each func(i int, bytes int64)) {
	var open []string
	var at []int
	for i, root := range roots {
		if guarded(u.expand, root) {
			each(i, 0)
			continue
		}
		open = append(open, root)
		at = append(at, i)
	}
	if len(open) > 0 {
		u.DiskUsage.Sizes(ctx, open, func(i int, bytes int64) { each(at[i], bytes) })
	}
}

func (u guardedUsage) Move(ctx context.Context, from, to string) error {
	if guarded(u.expand, from) || guarded(u.expand, to) {
		return errOtherAppsData
	}
	return u.DiskUsage.Move(ctx, from, to)
}

func (u guardedUsage) Erase(ctx context.Context, paths []string) error {
	for _, p := range paths {
		if guarded(u.expand, p) {
			return errOtherAppsData
		}
	}
	return u.DiskUsage.Erase(ctx, paths)
}
