package contracttest

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/remote/contract"
)

const Version = "v3.1.0"

var (
	// value: the far machine of every case: a Mac with jq to install, which
	// asks before it forgets, and two snapshots of its activity.
	Mini = machine.Profile{Ref: machine.Ref{Name: "mac-mini"}, Kind: machine.Computer, System: machine.System{OS: machine.MacOS}}
)

type InstallIn struct{ App string }

type InstallOut struct {
	App string
	// value: an error inside the answer, which must reach the caller as an error.
	Warning error
}

type ForgetIn struct{ App, Confirm string }

type Sample struct{ CPU int }

const Late = "jq installed, its shell completion did not"

func Far() *call.Registry {
	registry, err := call.NewRegistry([]call.Handler{
		call.Bind(call.Call[struct{}, InstallIn, InstallOut]{
			Name: "apps.install", Where: call.AtTarget, Applies: machine.Anywhere,
			Run: func(_ context.Context, _ struct{}, in InstallIn, p call.Progress) (InstallOut, error) {
				p.Step("installing "+in.App, "")
				p.Done("installing "+in.App, nil)
				return InstallOut{App: in.App, Warning: errors.New(Late)}, nil
			},
		}, struct{}{}),
		call.Bind(call.Call[struct{}, ForgetIn, struct{}]{
			Name: "apps.forget", Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.Forget,
			Run: func(_ context.Context, _ struct{}, in ForgetIn, _ call.Progress) (struct{}, error) {
				return struct{}{}, confirm.Require(confirm.For(confirm.Forget, machine.Computer), in.App, in.Confirm)
			},
		}, struct{}{}),
		call.BindWatch(call.Watch[struct{}, struct{}, Sample]{
			Name: "activity.read", Where: call.AtTarget, Applies: machine.Anywhere,
			Run: func(ctx context.Context, _ struct{}, _ struct{}, _ call.Progress) (<-chan call.Update[Sample], error) {
				out := make(chan Sample, 2)
				out <- Sample{CPU: 12}
				out <- Sample{CPU: 40}
				close(out)
				return call.Updates(ctx, out), nil
			},
		}, struct{}{}),
	})
	if err != nil {
		panic(err)
	}
	return registry
}

type EndpointSubject struct {
	Dialer contract.EndpointDialer
	// value: the Mac Dial is given, and the machine each call targets.
	Mac    string
	Target machine.Ref
	// value: from now on the far side does not answer.
	Down func()
}

type steps struct{ said []string }

func (s *steps) Step(task, _ string) { s.said = append(s.said, task) }
func (s *steps) Done(task string, _ error) {
	s.said = append(s.said, task+" done")
}

func Endpoint(t *testing.T, subject func(t *testing.T, far *call.Registry, profile machine.Profile, version string) EndpointSubject) {
	dial := func(t *testing.T) (EndpointSubject, contract.Endpoint) {
		t.Helper()
		s := subject(t, Far(), Mini, Version)
		ep, err := s.Dialer.Dial(t.Context(), s.Mac)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ep.Close() })
		return s, ep
	}
	t.Run("hello names the far binary's version and every call it serves", func(t *testing.T) {
		_, ep := dial(t)
		h, err := ep.Hello(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if h.Version != Version || !slices.Contains(h.Calls, "apps.install") || !slices.Contains(h.Calls, "activity.read") {
			t.Errorf("hello said %+v, want %s serving apps.install and activity.read", h, Version)
		}
	})
	t.Run("a call runs on the far side and its answer comes back whole, its warning still an error", func(t *testing.T) {
		s, ep := dial(t)
		in, _ := json.Marshal(InstallIn{App: "jq"})
		p := &steps{}
		raw, err := ep.Invoke(t.Context(), s.Target, "apps.install", in, p)
		if err != nil {
			t.Fatal(err)
		}
		var out InstallOut
		if err := call.Decode(raw, &out); err != nil {
			t.Fatal(err)
		}
		if out.App != "jq" || out.Warning == nil || out.Warning.Error() != Late {
			t.Errorf("the far install answered %+v, want jq with its warning %q", out, Late)
		}
		if !slices.Equal(p.said, []string{"installing jq", "installing jq done"}) {
			t.Errorf("the caller saw the steps %q, want the far side's two", p.said)
		}
	})
	t.Run("a refusal comes back with its code, so the caller words it", func(t *testing.T) {
		s, ep := dial(t)
		in, _ := json.Marshal(ForgetIn{App: "jq"})
		_, err := ep.Invoke(t.Context(), s.Target, "apps.forget", in, nil)
		var e *fail.Error
		if !errors.As(err, &e) || e.Type != fail.NeedsConfirmation || e.Code == "" || e.Subject != "jq" {
			t.Errorf("forgetting jq unconfirmed: %v, want a coded needs-confirmation naming jq", err)
		}
	})
	t.Run("a watch hands on each update as the far side sends it, then ends", func(t *testing.T) {
		s, ep := dial(t)
		updates, err := ep.Watch(t.Context(), s.Target, "activity.read", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		var cpu []int
		for u := range updates {
			var v Sample
			if u.Err != nil || call.Decode(u.Value, &v) != nil {
				t.Fatalf("an update failed: %v", u.Err)
			}
			cpu = append(cpu, v.CPU)
		}
		if !slices.Equal(cpu, []int{12, 40}) {
			t.Errorf("the watch handed %v, want [12 40]", cpu)
		}
	})
	t.Run("a far side that does not answer is unreachable, named", func(t *testing.T) {
		s := subject(t, Far(), Mini, Version)
		s.Down()
		var err error
		ep, err := s.Dialer.Dial(t.Context(), s.Mac)
		if err == nil {
			defer func() { _ = ep.Close() }()
			_, err = ep.Hello(t.Context())
			if err == nil {
				_, err = ep.Invoke(t.Context(), s.Target, "apps.install", nil, nil)
			}
		}
		var e *fail.Error
		if !errors.As(err, &e) || e.Type != fail.Unreachable || e.Code != call.Unreachable || e.Subject == "" {
			t.Errorf("a far side down: %v, want unreachable naming it", err)
		}
	})
}
