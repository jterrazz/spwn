package api_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
)

type recorded struct {
	io.Reader
	*strings.Builder
}

func (recorded) Close() error { return nil }

func sent(ctx context.Context, t *testing.T) api.Frame {
	t.Helper()
	var wrote strings.Builder
	c := api.NewClient(func(context.Context) (io.ReadWriteCloser, error) {
		return recorded{Reader: strings.NewReader(`{"v":2,"id":"1","out":{}}` + "\n"), Builder: &wrote}, nil
	})
	_ = c.Call(ctx, "ports.declare", machine.Ref{}, struct{}{}, nil, nil)
	var f api.Frame
	if err := json.Unmarshal([]byte(strings.SplitN(wrote.String(), "\n", 2)[0]), &f); err != nil {
		t.Fatalf("%q: %v", wrote.String(), err)
	}
	return f
}

func TestAFarCallRidesItsNextHopMarkedFar(t *testing.T) {
	if sent(t.Context(), t).Far {
		t.Fatal("a local call left marked far")
	}
	if !sent(call.WithFar(t.Context()), t).Far {
		t.Fatal("a far call left its mark behind")
	}
}

// value: what a connection wrote, read while its watch may still write a cancel.
type lines struct {
	mu   sync.Mutex
	text strings.Builder
}

func (l *lines) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.text.Write(p)
}

func (l *lines) first(n int) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	all := strings.Split(strings.TrimSpace(l.text.String()), "\n")
	return all[:min(n, len(all))]
}

func TestAConnectionCarriesTheFarMarkOnEveryFrameItSends(t *testing.T) {
	sends := func(ctx context.Context) (call, watch api.Frame) {
		wrote := &lines{}
		quiet, _ := io.Pipe()
		c := api.NewConn(quiet, wrote, func() error { return nil }, func(err error) error { return err }, "v0.1.0")
		ended, end := context.WithCancel(ctx)
		end()
		_, _ = c.Invoke(ended, machine.Ref{}, "devices.add", json.RawMessage(`{}`), nil)
		_, _ = c.Watch(ctx, machine.Ref{}, "devices.list", json.RawMessage(`{}`), nil)
		sent := wrote.first(2)
		if len(sent) != 2 || json.Unmarshal([]byte(sent[0]), &call) != nil || json.Unmarshal([]byte(sent[1]), &watch) != nil {
			t.Fatalf("the connection sent %q, want a call then a watch", sent)
		}
		return call, watch
	}
	if c, w := sends(t.Context()); c.Far || w.Far {
		t.Fatalf("a local call and watch left marked far: %v, %v", c.Far, w.Far)
	}
	if c, w := sends(call.WithFar(t.Context())); !c.Far || !w.Far {
		t.Fatalf("a far call and watch left their mark behind: %v, %v", c.Far, w.Far)
	}
}
