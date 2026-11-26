package wire

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

const applyWireVersion = "v0.21.0"

var (
	applyWireMac   = coremachine.Profile{Ref: coremachine.Ref{Name: "macbook-pro"}, Kind: coremachine.Computer, System: coremachine.System{OS: coremachine.MacOS}}
	applyWireGuest = coremachine.Profile{Ref: coremachine.Ref{Name: "web", On: "macbook-pro"}, Kind: coremachine.Workspace, System: coremachine.System{OS: coremachine.Linux}}
)

// value: this Mac's engine as a create reaches it, a connection of frames
// to a far side serving the guest's calls, as `station guest` serves them.
type guestEngine struct{ registry *call.Registry }

func (g guestEngine) Dial(ctx context.Context, _ string) (contract.Endpoint, error) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go func() {
		_ = serve.Stream(context.WithoutCancel(ctx), inR, outW, serve.Local(g.registry, applyWireGuest, applyWireVersion), func(err error) *fail.Error { return kit.Translate(err, "") })
		_ = outW.Close()
	}()
	return api.NewConn(outR, inW, inW.Close, func(err error) error { return err }, applyWireVersion), nil
}

// value: the guest's machinefile.apply as it ends at the bound the create
// hands it: the package carried out before the bound is applied, the one
// running there ends interrupted, and the run answers with both.
func cutAtItsBound(ctx context.Context, _ struct{}, in machinefilecalls.ApplyIn, _ call.Progress) (machinefile.Applied, error) {
	if in.Within <= 0 || !in.LeaveAsking {
		return machinefile.Applied{}, errors.New("the apply reached the guest without its bound or LeaveAsking")
	}
	bounded, cancel := context.WithTimeout(ctx, in.Within)
	defer cancel()
	jq := machinefile.Line{Step: machinefile.Step{Package: "jq", Kind: catalog.KindApp, Method: "apt"}, Action: machinefile.Install}
	rg := machinefile.Line{Step: machinefile.Step{Package: "ripgrep", Kind: catalog.KindApp, Method: "apt"}, Action: machinefile.Install}
	<-bounded.Done()
	return machinefile.Applied{Done: []machinefile.Done{
		{Line: jq, State: catalog.StateOK, Installed: true},
		{Line: rg, State: catalog.StateBroken, Reason: machinefile.Interrupted},
	}}, nil
}

// value: the create's router over a guest serving apply as given, its Mac
// serving this binary's apply, whose run is never asked.
func applyRouter(t *testing.T, apply call.Call[struct{}, machinefilecalls.ApplyIn, machinefile.Applied]) *remote.Router {
	t.Helper()
	guest, err := call.NewRegistry([]call.Handler{call.Bind(apply, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	here := apply
	here.Abilities = machinefilecalls.Apply.Abilities
	here.Run = func(context.Context, struct{}, machinefilecalls.ApplyIn, call.Progress) (machinefile.Applied, error) {
		t.Error("the create's apply ran on its Mac, never in its workspace")
		return machinefile.Applied{}, nil
	}
	mac, err := call.NewRegistry([]call.Handler{call.Bind(here, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	r := remote.NewRouter(mac, applyWireMac, remote.Reach{
		Engine: guestEngine{registry: guest}, Version: applyWireVersion,
		Locate: func(name string) (coremachine.Ref, bool) {
			if name == applyWireGuest.Ref.Name {
				return applyWireGuest.Ref, true
			}
			return coremachine.Ref{Name: name}, false
		},
	})
	t.Cleanup(r.Close)
	return r
}

func TestAGuestApplyCutAtItsBoundAnswersTheCreateWithWhatItDidOverTheWire(t *testing.T) {
	apply := call.Call[struct{}, machinefilecalls.ApplyIn, machinefile.Applied]{Name: machinefilecalls.Apply.Name, Where: call.AtTarget, Applies: coremachine.Anywhere, Abilities: machinefilecalls.Apply.Abilities, Run: cutAtItsBound}
	r := applyRouter(t, apply)
	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	got, err := machine{far: &late{router: r}}.applyAt(ctx, "web", "studio-linux", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("the apply cut at its bound answered the create %v; want what it did", err)
	}
	if len(got.Done) != 2 || got.Done[0].Line.Step.Package != "jq" || got.Done[0].State != catalog.StateOK || got.Done[1].Reason != machinefile.Interrupted {
		t.Errorf("the create read %+v; want jq installed, then ripgrep interrupted at the bound", got.Done)
	}
}

func TestAnOlderGuestIsRefusedTheCreatesApplyBeforeItRuns(t *testing.T) {
	// invariant: a guest whose station lists neither of the apply's
	// abilities decodes LeaveAsking and Within as absent, so the apply is
	// refused as far too old before any call reaches it.
	ran := false
	older := call.Call[struct{}, machinefilecalls.ApplyIn, machinefile.Applied]{Name: machinefilecalls.Apply.Name, Where: call.AtTarget, Applies: coremachine.Anywhere, Run: func(context.Context, struct{}, machinefilecalls.ApplyIn, call.Progress) (machinefile.Applied, error) {
		ran = true
		return machinefile.Applied{}, nil
	}}
	r := applyRouter(t, older)
	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	_, err := machine{far: &late{router: r}}.applyAt(ctx, "web", "studio-linux", time.Minute)
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != call.FarTooOld || refused.Detail["call"] != machinefilecalls.LeaveAskingAbility {
		t.Fatalf("an older guest answered the create %v; want it refused as far too old, lacking %s", err, machinefilecalls.LeaveAskingAbility)
	}
	if ran {
		t.Fatal("the older guest ran the apply it reads without its inputs")
	}
}
