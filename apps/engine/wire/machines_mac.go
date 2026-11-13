package wire

import (
	"context"
	"time"

	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type macSystem struct{ h local.System }

func (t macSystem) Now() time.Time { return t.h.Clock.Now() }
func (t macSystem) Sleep(ctx context.Context, d time.Duration) error {
	return t.h.Clock.Sleep(ctx, d)
}
func (t macSystem) Home() string     { return t.h.Env.Home() }
func (t macSystem) UID() int         { return t.h.Env.Euid() }
func (t macSystem) User() string     { return t.h.Env.User() }
func (t macSystem) Hostname() string { return t.h.Env.Hostname() }
func (t macSystem) Exists(path string) bool {
	_, err := t.h.FS.Stat(path)
	return err == nil
}
func (t macSystem) Remove(path string) error { return t.h.FS.Remove(path) }

func (t macSystem) Names(dir string) ([]string, error) {
	entries, err := t.h.FS.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

type machinesEngine struct{ macEngine }

func (d machinesEngine) GuestExec(ctx context.Context, vm, line, stdin string) (machinescalls.GuestRan, error) {
	out, err := d.macEngine.GuestExec(ctx, vm, line, stdin)
	return machinescalls.GuestRan{Stdout: out.Stdout, Stderr: out.Stderr, ExitCode: out.Code}, err
}
