package remote_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

func TestAWatchLetGoHereEndsItsProducerThere(t *testing.T) {
	ended := make(chan struct{})
	registry, err := call.NewRegistry([]call.Handler{call.BindWatch(call.Watch[struct{}, struct{}, contracttest.Sample]{
		Name: "activity.read", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(ctx context.Context, _ struct{}, _ struct{}, _ call.Progress) (<-chan call.Update[contracttest.Sample], error) {
			out := make(chan call.Update[contracttest.Sample])
			go func() {
				defer close(ended)
				defer close(out)
				select {
				case out <- call.Update[contracttest.Sample]{Value: contracttest.Sample{CPU: 1}}:
				case <-ctx.Done():
					return
				}
				<-ctx.Done()
			}()
			return out, nil
		},
	}, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go func() {
		_ = serve.Stream(context.WithoutCancel(t.Context()), inR, outW, serve.Local(registry, contracttest.Mini, "v3.3.0"), nil)
		_ = outW.Close()
	}()
	t.Cleanup(func() { _ = inW.Close() })
	c := api.NewConn(outR, inW, nil, nil, "v3.3.0")
	ctx, letGo := context.WithCancel(t.Context())
	updates, err := c.Watch(ctx, machine.Ref{}, "activity.read", json.RawMessage(`{}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	bound, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	select {
	case <-updates:
	case <-bound.Done():
		t.Fatal("the far watch never answered")
	}
	letGo()
	select {
	case <-ended:
	case <-bound.Done():
		t.Fatal("a watch let go of here still runs its producer there, its connection open")
	}
}

func TestAWatchThatAnswersAnErrorHasEndedItsProducerThere(t *testing.T) {
	ended := make(chan struct{})
	registry, err := call.NewRegistry([]call.Handler{call.BindWatch(call.Watch[struct{}, struct{}, contracttest.Sample]{
		Name: "activity.read", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(ctx context.Context, _ struct{}, _ struct{}, _ call.Progress) (<-chan call.Update[contracttest.Sample], error) {
			out := make(chan call.Update[contracttest.Sample])
			go func() {
				defer close(ended)
				defer close(out)
				for _, u := range []call.Update[contracttest.Sample]{
					{Value: contracttest.Sample{CPU: 1}},
					{Err: fail.New(fail.Refused, "machines.being-created", "", nil)},
				} {
					select {
					case out <- u:
					case <-ctx.Done():
						return
					}
				}
				<-ctx.Done()
			}()
			return out, nil
		},
	}, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go func() {
		_ = serve.Stream(context.WithoutCancel(t.Context()), inR, outW, serve.Local(registry, contracttest.Mini, "v3.3.0"), func(error) *fail.Error { return nil })
		_ = outW.Close()
	}()
	t.Cleanup(func() { _ = inW.Close() })
	c := api.NewConn(outR, inW, nil, nil, "v3.3.0")
	updates, err := c.Watch(t.Context(), machine.Ref{}, "activity.read", json.RawMessage(`{}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	bound, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	for range 2 {
		select {
		case <-updates:
		case <-bound.Done():
			t.Fatal("the far watch never answered")
		}
	}
	select {
	case <-ended:
	case <-bound.Done():
		t.Fatal("a watch that answered an error still runs its producer there, its connection open")
	}
}
