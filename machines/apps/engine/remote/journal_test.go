package remote_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote"
)

type kept struct{ lines, ran int }

func gestureRouter(t *testing.T, here machine.Profile, k *kept) *remote.Router {
	t.Helper()
	registry, err := call.NewRegistry([]call.Handler{call.Bind(call.Call[struct{}, installIn, installOut]{
		Name: "apps.remove", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, installIn, call.Progress) (installOut, error) {
			k.ran++
			return installOut{}, nil
		},
		Journal: func(in installIn, _ installOut, _ error) []journal.Line {
			return []journal.Line{{Text: in.App + " removed"}}
		},
	}, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	return remote.NewRouter(registry, here, remote.Reach{Kept: counted(k)})
}

func counted(k *kept) call.Keeper {
	return func(_ context.Context, _ machine.Ref, m call.Meta, in, out any, err error) {
		k.lines += len(m.Journal(in, out, err))
	}
}

func TestAGestureIsKeptOnceByTheRouterOfTheMacThatRunsIt(t *testing.T) {
	k := &kept{}
	r := gestureRouter(t, laptop, k)
	if err := r.Invoke(t.Context(), machine.Ref{}, "apps.remove", installIn{App: "jq"}, &installOut{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Serve(t.Context(), laptop.Ref, "apps.remove", json.RawMessage(`{"App":"jq"}`), nil); err != nil {
		t.Fatal(err)
	}
	if k.ran != 2 || k.lines != 2 {
		t.Errorf("ran %d, kept %d lines; want each of the two runs kept once", k.ran, k.lines)
	}
}

func TestARouterInsideAVMKeepsNothing(t *testing.T) {
	k := &kept{}
	if err := gestureRouter(t, linux1, k).Invoke(t.Context(), machine.Ref{}, "apps.remove", installIn{App: "jq"}, &installOut{}, nil); err != nil {
		t.Fatal(err)
	}
	if k.ran != 1 || k.lines != 0 {
		t.Errorf("ran %d, kept %d lines; want it run and nothing kept inside a VM", k.ran, k.lines)
	}
}

func TestAGestureByAnAuthorThatIsNoSessionIsRefusedUnrun(t *testing.T) {
	for by, refused := range map[string]bool{"Lot_Shell": true, "the night": false, "station": false, "you": false, "review-bot": false} {
		k := &kept{}
		err := gestureRouter(t, laptop, k).Invoke(call.WithBy(t.Context(), by), machine.Ref{}, "apps.remove", installIn{App: "jq"}, &installOut{}, nil)
		var e *fail.Error
		if got := errors.As(err, &e) && e.Code == call.BadInput && e.Detail["in"] == "by"; got != refused || refused && k.ran != 0 {
			t.Errorf("by %q: %v, ran %d; want refused %v, never run when refused", by, err, k.ran, refused)
		}
	}
}

func TestAGestureRunInsideAnotherKeepsNothingOfItsOwn(t *testing.T) {
	k := &kept{}
	var r *remote.Router
	removed := func(in installIn, _ installOut, _ error) []journal.Line {
		return []journal.Line{{Text: in.App + " removed"}}
	}
	registry, err := call.NewRegistry([]call.Handler{
		call.Bind(call.Call[struct{}, installIn, installOut]{
			Name: "apps.remove", Where: call.AtTarget, Applies: machine.Anywhere, Journal: removed,
			Run: func(context.Context, struct{}, installIn, call.Progress) (installOut, error) {
				k.ran++
				return installOut{}, nil
			},
		}, struct{}{}),
		call.Bind(call.Call[struct{}, installIn, installOut]{
			Name: "apps.install", Where: call.AtTarget, Applies: machine.Anywhere, Journal: removed,
			Run: func(ctx context.Context, _ struct{}, in installIn, p call.Progress) (installOut, error) {
				return installOut{}, r.Invoke(ctx, machine.Ref{}, "apps.remove", in, &installOut{}, p)
			},
		}, struct{}{}),
	})
	if err != nil {
		t.Fatal(err)
	}
	r = remote.NewRouter(registry, laptop, remote.Reach{Kept: counted(k)})
	if err := r.Invoke(t.Context(), machine.Ref{}, "apps.install", installIn{App: "jq"}, &installOut{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Serve(t.Context(), laptop.Ref, "apps.install", json.RawMessage(`{"App":"jq"}`), nil); err != nil {
		t.Fatal(err)
	}
	if k.ran != 2 || k.lines != 2 {
		t.Errorf("inner ran %d, kept %d lines; want the inner gesture run twice and only the two outer runs kept", k.ran, k.lines)
	}
}
