package api_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
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
