package process

import (
	"context"
	"errors"
	"sync"
	"time"
)

type clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

// value: a clock that reads At and moves it by every sleep.
type stepClock struct{ At time.Time }

func (c *stepClock) Now() time.Time { return c.At }

func (c *stepClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.At = c.At.Add(d)
	return nil
}

// value: a Mac of a module test: no file, every socket idle.
type fakeMac struct{ clock }

func (*fakeMac) Home() string                        { return "/home/u" }
func (*fakeMac) ReadFile(string) ([]byte, error)     { return nil, errors.New("no file in a module test") }
func (*fakeMac) Exists(string) bool                  { return false }
func (*fakeMac) MkdirAll(string) error               { return nil }
func (*fakeMac) Remove(string) error                 { return nil }
func (*fakeMac) ListenUnix(string) (Listener, error) { return newIdle(), nil }
func (*fakeMac) Join(a, b Stream)                    { _, _ = a.Close(), b.Close() }
func (*fakeMac) Reap(string) int                     { return 0 }

func (*fakeMac) ListenLoopback(int) (Listener, error) { return newIdle(), nil }

type idleListener struct {
	once sync.Once
	done chan struct{}
}

func newIdle() *idleListener { return &idleListener{done: make(chan struct{})} }

func (l *idleListener) Accept() (Stream, error) {
	<-l.done
	return nil, errors.New("closed")
}

func (l *idleListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *idleListener) Port() int { return 0 }
