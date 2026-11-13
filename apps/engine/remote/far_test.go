package remote_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
)

const mini = "mac-mini"

// value: the laptop, whose Router reaches the mini (a fake second Mac) and
// its own VM linux-1 through its engine (another fake, to tell them apart).
func laptopReaching(t *testing.T, miniVersion string) (r *remote.Router, macs, engine *fake.Dialer, ran *int) {
	t.Helper()
	ran = new(int)
	registry, err := call.NewRegistry(installs(ran))
	if err != nil {
		t.Fatal(err)
	}
	macs, engine = fake.New(), fake.New()
	macs.Serve(mini, fake.Mac{Registry: contracttest.Far(), Profile: contracttest.Mini, Version: miniVersion})
	engine.Serve(laptop.Ref.Name, fake.Mac{Registry: contracttest.Far(), Profile: linux1, Version: "v3.1.0"})
	r = remote.NewRouter(registry, laptop, remote.Reach{
		Macs: macs, Engine: engine, Version: "v3.1.0",
		Locate: func(name string) (machine.Ref, bool) {
			switch name {
			case linux1.Ref.Name:
				return linux1.Ref, true
			case mini:
				return machine.Ref{Name: name}, true
			}
			return machine.Ref{Name: name}, false
		},
		Holder: func(_ context.Context, vm string) (string, error) {
			if vm == infra.Ref.Name {
				return mini, nil
			}
			return "", errNoMachine
		},
	})
	t.Cleanup(r.Close)
	return r, macs, engine, ran
}

func TestOnAnotherMacTheCallRunsThereAndItsAnswerIsWholeHere(t *testing.T) {
	r, _, _, ran := laptopReaching(t, "v3.1.0")
	out, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, farInstall, contracttest.InstallIn{App: "jq"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.App != "jq" || out.Warning == nil || out.Warning.Error() != contracttest.Late {
		t.Errorf("installing jq on the mini answered %+v, want jq and its warning as an error", out)
	}
	if *ran != 0 {
		t.Errorf("the install ran %d times on the laptop, want only on the mini", *ran)
	}
}

func TestAVMOfThisMacIsReachedThroughThisMacsEngine(t *testing.T) {
	r, macs, _, _ := laptopReaching(t, "v3.1.0")
	macs.Down(mini)
	out, err := call.Invoke(t.Context(), r, machine.Ref{Name: "linux-1"}, farInstall, contracttest.InstallIn{App: "jq"}, nil)
	if err != nil || out.App != "jq" {
		t.Errorf("installing jq on linux-1: %+v, %v; want it run through the laptop's engine", out, err)
	}
}

func TestAFarBinaryWithoutTheCallIsTooOldAndSaysItsVersion(t *testing.T) {
	r, _, _, _ := laptopReaching(t, "v3.0.0")
	_, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, call.Call[struct{}, struct{}, struct{}]{Name: "ports.declare"}, struct{}{}, nil)
	var e *fail.Error
	if !errors.As(err, &e) || e.Type != fail.FarTooOld || e.Subject != mini || e.Detail["version"] != "v3.0.0" || e.Detail["call"] != "ports.declare" {
		t.Errorf("ports.declare on a mini that lacks it: %v, want far-too-old naming the mini, v3.0.0 and the call", err)
	}
}

func TestAMacThatDoesNotAnswerIsUnreachableNamed(t *testing.T) {
	r, macs, _, _ := laptopReaching(t, "v3.1.0")
	macs.Down(mini)
	_, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, farInstall, contracttest.InstallIn{App: "jq"}, nil)
	var e *fail.Error
	if !errors.As(err, &e) || e.Type != fail.Unreachable || e.Subject != mini {
		t.Errorf("a call to the mini offline: %v, want unreachable naming it", err)
	}
}

func TestAServerRunsAFrameForItsOwnMachineHereAndForItsVMOverItsEngine(t *testing.T) {
	r, _, _, ran := laptopReaching(t, "v3.1.0")
	if _, err := r.Serve(t.Context(), machine.Ref{Name: "laptop-alias"}, "apps.install-at-target", []byte(`{"App":"jq"}`), nil); err != nil || *ran != 1 {
		t.Errorf("a frame for the laptop itself: %v, ran %d; want it run here once", err, *ran)
	}
	v, err := r.Serve(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", []byte(`{"App":"jq"}`), nil)
	var out contracttest.InstallOut
	if raw, ok := v.(json.RawMessage); err != nil || !ok || call.Decode(raw, &out) != nil || out.App != "jq" {
		t.Errorf("a frame for linux-1: %v, %v; want it answered by linux-1 over the engine", v, err)
	}
}

func TestAVMOfAnotherMacIsReachedThroughThatMacsEngineNeverByItsOwnName(t *testing.T) {
	r, macs, engine, _ := laptopReaching(t, "v3.1.0")
	macs.Serve(infra.Ref.Name, fake.Mac{Registry: contracttest.Far(), Profile: infra, Version: "v3.1.0"})
	engine.Down(laptop.Ref.Name)
	out, err := call.Invoke(t.Context(), r, machine.Ref{Name: infra.Ref.Name}, farInstall, contracttest.InstallIn{App: "jq"}, nil)
	if err != nil || out.App != "jq" {
		t.Fatalf("installing jq on infra, a VM of the mini: %+v, %v; want it run through the mini's engine", out, err)
	}
	if got := macs.Dialed(); !slices.Equal(got, []string{mini}) {
		t.Errorf("reaching infra dialed %q, want the mini alone, never infra by its own name", got)
	}
}

func TestANameNoMacHoldsIsRefusedWithoutDiallingIt(t *testing.T) {
	r, macs, _, _ := laptopReaching(t, "v3.1.0")
	_, err := call.Invoke(t.Context(), r, machine.Ref{Name: "nope"}, farInstall, contracttest.InstallIn{App: "jq"}, nil)
	if !errors.Is(err, errNoMachine) {
		t.Errorf("a call to nope: %v, want the holder's no-machine answer", err)
	}
	if got := macs.Dialed(); len(got) != 0 {
		t.Errorf("a call to nope dialed %q, want nothing dialed", got)
	}
}

func TestAMacUpgradedWhileTheConnectionLivesServesTheCallItLacked(t *testing.T) {
	r, macs, _, _ := laptopReaching(t, "v3.0.0")
	if _, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, portsDeclare, struct{}{}, nil); !isType(err, fail.FarTooOld) {
		t.Fatalf("ports.declare on the mini at v3.0.0: %v, want far-too-old", err)
	}
	macs.Serve(mini, fake.Mac{Registry: withPortsDeclare(t), Profile: contracttest.Mini, Version: "v3.1.0"})
	if _, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, portsDeclare, struct{}{}, nil); err != nil {
		t.Errorf("the same ports.declare after the mini is upgraded to v3.1.0: %v, want it declared", err)
	}
}

func TestAMacAsleepAtTheFirstCallAnswersOnceItWakes(t *testing.T) {
	r, macs, _, _ := laptopReaching(t, "v3.1.0")
	macs.Down(mini)
	if _, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, farInstall, contracttest.InstallIn{App: "jq"}, nil); !isType(err, fail.Unreachable) {
		t.Fatalf("a call to the mini asleep: %v, want unreachable", err)
	}
	macs.Serve(mini, fake.Mac{Registry: contracttest.Far(), Profile: contracttest.Mini, Version: "v3.1.0"})
	if out, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, farInstall, contracttest.InstallIn{App: "jq"}, nil); err != nil || out.App != "jq" {
		t.Errorf("the same call once the mini wakes: %+v, %v; want jq installed there", out, err)
	}
}

func TestAMacWhoseLinkDroppedIsDialedAgainAndOneLiveLinkServesEveryCall(t *testing.T) {
	r, macs, _, _ := laptopReaching(t, "v3.1.0")
	install := func() error {
		_, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, farInstall, contracttest.InstallIn{App: "jq"}, nil)
		return err
	}
	if err := errors.Join(install(), install()); err != nil {
		t.Fatal(err)
	}
	macs.Down(mini)
	if err := install(); !isType(err, fail.Unreachable) {
		t.Fatalf("a call after the mini's link dropped: %v, want unreachable", err)
	}
	macs.Serve(mini, fake.Mac{Registry: contracttest.Far(), Profile: contracttest.Mini, Version: "v3.1.0"})
	if err := install(); err != nil {
		t.Errorf("a call once the mini answers again: %v, want it run there", err)
	}
	if got := macs.Dialed(); len(got) != 2 {
		t.Errorf("the mini was dialed %d times, want 2: once for the two calls that shared a live link, once after it dropped", len(got))
	}
}

func TestACallTheLinkLacksNeverCutsTheCallsStillRidingIt(t *testing.T) {
	r, _, _, _ := laptopReaching(t, "v3.0.0")
	updates, err := r.Watch(t.Context(), machine.Ref{Name: mini}, "activity.read", struct{}{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, portsDeclare, struct{}{}, nil); !isType(err, fail.FarTooOld) {
		t.Fatalf("ports.declare on the mini at v3.0.0: %v, want far-too-old", err)
	}
	read := 0
	for u := range updates {
		if u.Err != nil {
			t.Fatalf("the mini's activity, open while ports.declare was refused: %v after %d samples, want both", u.Err, read)
		}
		read++
	}
	if read != 2 {
		t.Errorf("the mini's activity gave %d samples, want 2", read)
	}
}

func isType(err error, want fail.Type) bool {
	var e *fail.Error
	return errors.As(err, &e) && e.Type == want
}

var portsDeclare = call.Call[struct{}, struct{}, struct{}]{Name: "ports.declare"}

// value: the mini's calls once upgraded to v3.1.0, which adds ports.declare.
func withPortsDeclare(t *testing.T) *call.Registry {
	t.Helper()
	handlers := []call.Handler{call.Bind(call.Call[struct{}, struct{}, struct{}]{
		Name: "ports.declare", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (struct{}, error) { return struct{}{}, nil },
	}, struct{}{})}
	registry, err := call.NewRegistry(handlers)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

var errNoMachine = errors.New(`no machine named "nope"`)

var farInstall = call.Call[struct{}, contracttest.InstallIn, contracttest.InstallOut]{Name: "apps.install"}
