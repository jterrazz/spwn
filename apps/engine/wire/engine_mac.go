package wire

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

// value: this Mac as the engine process reads it, over platform/local.
type engineMac struct{ local.System }

var _ process.Mac = engineMac{}

func (m engineMac) Now() time.Time                                   { return m.Clock.Now() }
func (m engineMac) Sleep(ctx context.Context, d time.Duration) error { return m.Clock.Sleep(ctx, d) }
func (m engineMac) Home() string                                     { return m.Env.Home() }
func (m engineMac) ReadFile(path string) ([]byte, error)             { return m.FS.ReadFile(path) }
func (m engineMac) MkdirAll(path string) error                       { return m.FS.MkdirAll(path, 0o700) }
func (m engineMac) Remove(path string) error                         { return m.FS.Remove(path) }
func (m engineMac) Join(a, b process.Stream)                         { m.Sockets.Join(a, b) }
func (m engineMac) Reap(dir string) int                              { return m.Run.Reap(dir) }

func (m engineMac) Exists(path string) bool {
	_, err := m.FS.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func (m engineMac) ListenUnix(path string) (process.Listener, error) {
	l, err := m.Sockets.ListenUnix(path)
	if err != nil {
		return nil, err
	}
	return engineListener{l}, nil
}

func (m engineMac) ListenLoopback(port int) (process.Listener, error) {
	l, err := m.Sockets.ListenTCP(ports.Loopback, port)
	if err != nil {
		return nil, err
	}
	return engineListener{l}, nil
}

type engineListener struct{ local.Listener }

func (l engineListener) Accept() (process.Stream, error) {
	c, err := l.Listener.Accept()
	if c == nil {
		return nil, err
	}
	return c, err
}
