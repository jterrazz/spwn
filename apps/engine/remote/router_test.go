package remote_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

type installIn struct{ App string }

type installOut struct {
	App string
	// value: an error inside the result, which must reach the leaf as an error.
	Warning error
}

var (
	laptop = machine.Profile{Ref: machine.Ref{Name: "jterrazz-macbook-pro"}, Kind: machine.Computer, System: machine.System{OS: machine.MacOS}}
	infra  = machine.Profile{Ref: machine.Ref{Name: "infra", On: "jterrazz-mac-mini"}, Kind: machine.Server, System: machine.System{OS: machine.Linux}}
	linux1 = machine.Profile{Ref: machine.Ref{Name: "linux-1", On: "jterrazz-macbook-pro"}, Kind: machine.Workspace, System: machine.System{OS: machine.Linux}}
)

func onMacOS(p machine.Profile) machine.Verdict {
	if p.System.OS == machine.Linux {
		return machine.No(p.Ref.Name + " runs Linux: Mac App Store apps live on macOS only")
	}
	return machine.Yes
}

var errLate = errors.New("jq installed, its shell completion did not")

// value: an install in each Where, so a router that sends one Where elsewhere is seen.
func installs(ran *int) []call.Handler {
	run := func(_ context.Context, _ struct{}, in installIn, _ call.Progress) (installOut, error) {
		*ran++
		return installOut{App: in.App, Warning: errLate}, nil
	}
	out := make([]call.Handler, 0, 4)
	for _, where := range []call.Where{call.AtTarget, call.InEngine, call.InProcess} {
		out = append(out, call.Bind(call.Call[struct{}, installIn, installOut]{
			Name: "apps.install-" + string(where), Section: section.Apps, Where: where, Applies: onMacOS, Run: run,
		}, struct{}{}))
	}
	out = append(out, call.BindWatch(call.Watch[struct{}, installIn, installOut]{
		Name: "apps.list", Section: section.Apps, Where: call.AtTarget, Applies: onMacOS,
		Run: func(ctx context.Context, _ struct{}, _ installIn, _ call.Progress) (<-chan call.Update[installOut], error) {
			*ran++
			return call.Updates(ctx, make(chan installOut)), nil
		},
	}, struct{}{}))
	return out
}

func routerOn(t *testing.T, here machine.Profile, ran *int) *remote.Router {
	t.Helper()
	registry, err := call.NewRegistry(installs(ran))
	if err != nil {
		t.Fatal(err)
	}
	return remote.NewRouter(registry, here, remote.Reach{})
}

func TestEveryCallForThisMachineRunsInThisProcessWithItsValuesWhole(t *testing.T) {
	for _, where := range []call.Where{call.AtTarget, call.InEngine, call.InProcess} {
		for _, target := range []machine.Ref{{}, laptop.Ref} {
			ran := 0
			var out installOut
			err := routerOn(t, laptop, &ran).Invoke(t.Context(), target, "apps.install-"+string(where), installIn{App: "jq"}, &out, nil)
			if err != nil || ran != 1 || out.App != "jq" || !errors.Is(out.Warning, errLate) {
				t.Errorf("installing jq %s on %q: %+v, %v, ran %d times; want it run here once, its warning still an error", where, target.Name, out, err, ran)
			}
		}
	}
}

func TestACallRunInThisProcessPassesTheGateOfAFrame(t *testing.T) {
	for _, c := range []struct {
		gesture string
		here    machine.Profile
		want    string
	}{
		{"a server has no Apps section", infra, "apps"},
		{"Mac App Store apps on a Linux VM", linux1, "linux-1 runs Linux: Mac App Store apps live on macOS only"},
	} {
		t.Run(c.gesture, func(t *testing.T) {
			ran := 0
			r := routerOn(t, c.here, &ran)
			var out installOut
			err := r.Invoke(t.Context(), machine.Ref{}, "apps.install-in-process", installIn{App: "keynote"}, &out, nil)
			_, watchErr := r.Watch(t.Context(), machine.Ref{}, "apps.list", installIn{}, nil)
			for _, err := range []error{err, watchErr} {
				var e *fail.Error
				if !errors.As(err, &e) || e.Code != call.NotForThisMachine || (e.Detail["section"] != c.want && e.Detail["why"] != c.want) {
					t.Errorf("got %v, want not-for-this-machine saying %q", err, c.want)
				}
			}
			if ran != 0 {
				t.Errorf("the refused call ran %d times", ran)
			}
		})
	}
}

type pipe struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func (p pipe) Close() error {
	for _, c := range p.closers {
		_ = c.Close()
	}
	return nil
}

type updated struct{ App string }

// value: an update in each Where, its result plain data, counted where it ran.
func updates(ran *int) *call.Registry {
	out := make([]call.Handler, 0, 3)
	for _, where := range []call.Where{call.AtTarget, call.InEngine, call.InProcess} {
		out = append(out, call.Bind(call.Call[struct{}, installIn, updated]{
			Name: "apps.update-" + string(where), Section: section.Apps, Where: where, Applies: onMacOS,
			Run: func(_ context.Context, _ struct{}, in installIn, _ call.Progress) (updated, error) {
				*ran++
				return updated(in), nil
			},
		}, struct{}{}))
	}
	registry, _ := call.NewRegistry(out)
	return registry
}

// value: this Mac's engine: a server of the same calls over a pipe.
func engineOf(t *testing.T, ran *int) call.Router {
	t.Helper()
	s := serve.Server{Calls: updates(ran).Lookup, ThisMachine: laptop}
	return remote.Endpoint(api.NewClient(func(context.Context) (io.ReadWriteCloser, error) {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		go s.Conn(t.Context(), pipe{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
		return pipe{Reader: outR, Writer: inW, closers: []io.Closer{outR, inW}}, nil
	}))
}

func TestACallThatNeedsTheEngineRunsInThisMacsEngineAndTheOthersHere(t *testing.T) {
	here, there := 0, 0
	r := remote.NewRouter(updates(&here), laptop, remote.Reach{}).WithEngine(engineOf(t, &there))
	for _, where := range []call.Where{call.AtTarget, call.InEngine, call.InProcess} {
		var out updated
		if err := r.Invoke(t.Context(), laptop.Ref, "apps.update-"+string(where), installIn{App: "jq"}, &out, nil); err != nil || out.App != "jq" {
			t.Fatalf("updating jq %s: %+v, %v", where, out, err)
		}
	}
	if here != 2 || there != 1 {
		t.Errorf("ran %d here and %d in the engine, want the in-engine call there and the two others here", here, there)
	}
}
