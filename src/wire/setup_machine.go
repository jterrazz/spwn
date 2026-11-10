package wire

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/jterrazz/studio-station/src/platform/local"
	setupcalls "github.com/jterrazz/studio-station/src/setup/calls"
)

type setupMachine struct{ h local.System }

func (s setupMachine) Home() string     { return s.h.Env.Home() }
func (s setupMachine) User() string     { return s.h.Env.User() }
func (s setupMachine) Hostname() string { return s.h.Env.Hostname() }
func (s setupMachine) Sleep(ctx context.Context, d time.Duration) error {
	return s.h.Clock.Sleep(ctx, d)
}

func (s setupMachine) ReadFile(path string) (data []byte, found bool, err error) {
	data, err = s.h.FS.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func (s setupMachine) WriteFile(path string, data []byte, mode uint32) error {
	return s.h.FS.WriteFile(path, data, fs.FileMode(mode))
}

func (s setupMachine) MkdirAll(path string, mode uint32) error {
	return s.h.FS.MkdirAll(path, fs.FileMode(mode))
}

func (s setupMachine) Mode(path string, otherwise uint32) uint32 {
	if info, err := s.h.FS.Stat(path); err == nil {
		return uint32(info.Mode().Perm())
	}
	return otherwise
}

func (s setupMachine) Exists(path string) bool {
	_, err := s.h.FS.Stat(path)
	return err == nil
}

func (s setupMachine) IsDir(path string) (dir, found bool) {
	info, err := s.h.FS.Stat(path)
	if err != nil {
		return false, false
	}
	return info.IsDir(), true
}

func (s setupMachine) Entries(dir string) ([]setupcalls.Entry, bool, error) {
	entries, err := s.h.FS.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	out := make([]setupcalls.Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, setupcalls.Entry{Name: e.Name(), Dir: e.IsDir()})
	}
	return out, true, nil
}

func (s setupMachine) Glob(pattern string) []string {
	matches, _ := s.h.FS.Glob(pattern)
	return matches
}

func (s setupMachine) Remove(path string) (bool, error) {
	err := s.h.FS.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (s setupMachine) Rename(from, to string) (bool, error) {
	err := s.h.FS.Rename(from, to)
	return errors.Is(err, local.ErrCrossDevice), err
}

func (s setupMachine) Output(ctx context.Context, name string, args []string, budget time.Duration) (stdout string, code int, err error) {
	out, err := s.h.Run.Output(ctx, local.Cmd{Name: name, Args: args, Timeout: budget})
	return out.Stdout, out.ExitCode, err
}

func (s setupMachine) Start(name string, args, env []string, log string) error {
	_, err := s.h.Run.Start(local.Cmd{Name: name, Args: args, Env: env}, log)
	return err
}

func (s setupMachine) Alive(pid int) bool      { return s.h.Run.Alive(pid) }
func (s setupMachine) Terminate(pid int) error { return s.h.Run.Signal(pid, local.SigTerm) }
func (s setupMachine) Kill(pid int) error      { return s.h.Run.Signal(pid, local.SigKill) }
