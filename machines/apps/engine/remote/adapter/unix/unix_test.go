package unix_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/unix"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

// value: this Mac's engine, whose one connection to the VM runs the line it
// is given there: the VM's `station guest serve --stdio`, served here.
type engine struct {
	vm serve.Calls
	// value: the VM's server from the second exec on: its binary replaced
	// between two execs; nil keeps vm.
	then  serve.Calls
	down  bool
	lines []string
	stdin []string
	calls []string
}

func (e *engine) GuestExec(ctx context.Context, vm, name, line, stdin string) (stdout string, code int, err error) {
	e.lines, e.stdin, e.calls = append(e.lines, line), append(e.stdin, stdin), append(e.calls, name)
	if e.down {
		return "", 0, fail.New(fail.Unreachable, call.Unreachable, vm, nil)
	}
	guest := e.vm
	if e.then != nil && len(e.lines) > 1 {
		guest = e.then
	}
	var out bytes.Buffer
	_ = serve.Stream(ctx, strings.NewReader(stdin), &out, guest, func(err error) *fail.Error {
		var f *fail.Error
		if errors.As(err, &f) {
			return f
		}
		return &fail.Error{Message: err.Error(), Exit: fail.ExitFailure}
	})
	return out.String(), 0, nil
}

func TestAVMOfThisMacReachedThroughItsEngineKeepsTheEndpointContract(t *testing.T) {
	contracttest.Endpoint(t, func(_ *testing.T, far *call.Registry, profile machine.Profile, version string) contracttest.EndpointSubject {
		e := &engine{vm: serve.Local(far, profile, version)}
		return contracttest.EndpointSubject{
			Dialer: unix.New(e, e.vm.Hello, nil, nil), Mac: "macbook-pro",
			Target: machine.Ref{Name: "linux-1", On: "macbook-pro"},
			Down:   func() { e.down = true },
		}
	})
}

func TestAVMIsAskedForItsOwnCallsOnlyThroughTheGuestServer(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v0.21.0-3-gabcdef1")}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{Version: contracttest.Version} }, nil, nil).Dial(t.Context(), "macbook-pro")
	_, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "ports.declare", nil, nil)
	var f *fail.Error
	if !errors.As(err, &f) || f.Type != fail.FarTooOld || f.Detail["version"] != "v0.21.0-3-gabcdef1" || f.Subject != "linux-1" {
		t.Errorf("asking linux-1 on v0.21.0-3-gabcdef1 for ports.declare: %v, want far-too-old naming linux-1 and v0.21.0-3-gabcdef1", err)
	}
	if len(e.lines) != 2 || e.lines[0] != unix.Guest || e.lines[1] != unix.Guest {
		t.Errorf("the engine ran %q inside the VM, want only %q", e.lines, unix.Guest)
	}
	if len(e.calls) != 2 || e.calls[0] != "" || e.calls[1] != "ports.declare" {
		t.Errorf("the engine was told the exchanges carried %q, want the Hello alone then ports.declare", e.calls)
	}
}

func TestACallsInputTravelsOnTheGuestsStdinNeverOnItsCommandLine(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v0.21.0-3-gabcdef1")}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{Version: contracttest.Version} }, nil, nil).Dial(t.Context(), "macbook-pro")
	text := "yes, merge it; then `rm -rf build` && say $HOME"
	in, _ := json.Marshal(contracttest.InstallIn{App: text})
	_, _ = ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", in, nil)
	if len(e.lines) != 2 || e.lines[0] != unix.Guest || e.lines[1] != unix.Guest {
		t.Fatalf("the engine ran %q inside the VM, want only %q", e.lines, unix.Guest)
	}
	sent, _ := json.Marshal(text)
	if !strings.Contains(e.stdin[1], strings.Trim(string(sent), `"`)) {
		t.Errorf("the input is not in the guest's stdin %q", e.stdin[1])
	}
}

func TestAGuestOnAnotherReleaseIsRefusedBeforeTheRequestReachesIt(t *testing.T) {
	for _, c := range []struct {
		version string
		typ     fail.Type
		code    fail.Code
	}{
		{"v0.0.9", fail.FarTooOld, call.FarTooOld},
		{"v0.20.0", fail.FarTooOld, call.FarTooOld},
		{"v3.2.0", fail.FarTooOld, call.FarTooOld},
		{"v0.22.0", fail.Refused, call.CallerTooOld},
	} {
		e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, c.version)}
		ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{Version: contracttest.Version} }, imageOf, nil).Dial(t.Context(), "macbook-pro")
		in, _ := json.Marshal(contracttest.InstallIn{App: "jq"})
		_, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", in, nil)
		var f *fail.Error
		if !errors.As(err, &f) || f.Type != c.typ || f.Code != c.code || f.Subject != "linux-1" || f.Detail["version"] != c.version {
			t.Errorf("installing jq on linux-1 at %s: %v, want %s naming linux-1 and %s", c.version, err, c.code, c.version)
		}
		if c.typ == fail.FarTooOld && f != nil && f.Detail["image"] != "base-ubuntu" {
			t.Errorf("the refusal of linux-1 at %s names image %q, want the image it was made from", c.version, f.Detail["image"])
		}
		if len(e.stdin) != 1 || strings.Contains(e.stdin[0], "apps.install") {
			t.Errorf("the guest at %s was sent %q, want its hello alone", c.version, e.stdin)
		}
	}
}

func TestAGuestOnThisMacsReleaseIsServed(t *testing.T) {
	for _, version := range []string{contracttest.Version, contracttest.Version + "-2-gabcdef1-dirty"} {
		e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, version)}
		ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{Version: contracttest.Version} }, imageOf, nil).Dial(t.Context(), "macbook-pro")
		in, _ := json.Marshal(contracttest.InstallIn{App: "jq"})
		if _, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", in, nil); err != nil {
			t.Errorf("installing jq on linux-1 at %s: %v, want it served", version, err)
		}
	}
}

func imageOf(vm string) string {
	if vm == "linux-1" {
		return "base-ubuntu"
	}
	return ""
}

func TestAGuestWhoseBinaryTurnsToAnOlderReleaseBetweenTwoExecsIsRefusedAtTheSecond(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v0.21.0"), then: serve.Local(contracttest.Far(), contracttest.Mini, "v0.0.9")}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{Version: contracttest.Version} }, imageOf, nil).Dial(t.Context(), "macbook-pro")
	in, _ := json.Marshal(contracttest.InstallIn{App: "jq"})
	_, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", in, nil)
	var f *fail.Error
	if !errors.As(err, &f) || f.Type != fail.FarTooOld || f.Detail["version"] != "v0.0.9" || f.Detail["image"] != "base-ubuntu" {
		t.Errorf("installing jq on linux-1 whose station became v0.0.9 after its first hello: %v, want far-too-old naming v0.0.9 and its image", err)
	}
}

func TestAGuestThatLacksAnAbilityTheInputAsksIsRefusedBeforeTheRequestReachesIt(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v0.21.0")}
	asks := func(name string, _ json.RawMessage) []string {
		if name == "apps.install" {
			return []string{"apps.install.pinned"}
		}
		return nil
	}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{Version: contracttest.Version} }, imageOf, asks).Dial(t.Context(), "macbook-pro")
	in, _ := json.Marshal(contracttest.InstallIn{App: "jq"})
	_, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", in, nil)
	var f *fail.Error
	if !errors.As(err, &f) || f.Type != fail.FarTooOld || f.Subject != "linux-1" || f.Detail["call"] != "apps.install.pinned" || f.Detail["image"] != "base-ubuntu" {
		t.Errorf("installing jq with an ability linux-1 lacks: %v, want far-too-old naming linux-1, the ability and its image", err)
	}
	if len(e.stdin) != 1 || strings.Contains(e.stdin[0], "apps.install") {
		t.Errorf("the guest that lacks the ability was sent %q, want its hello alone", e.stdin)
	}
}
