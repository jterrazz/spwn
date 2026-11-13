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
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
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
}

func (e *engine) GuestExec(ctx context.Context, vm, line, stdin string) (stdout string, code int, err error) {
	e.lines, e.stdin = append(e.lines, line), append(e.stdin, stdin)
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
			Dialer: unix.New(e, e.vm.Hello, nil), Mac: "jterrazz-macbook-pro",
			Target: machine.Ref{Name: "linux-1", On: "jterrazz-macbook-pro"},
			Down:   func() { e.down = true },
		}
	})
}

func TestAVMIsAskedForItsOwnCallsOnlyThroughTheGuestServer(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v3.2.0")}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{} }, nil).Dial(t.Context(), "jterrazz-macbook-pro")
	_, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "ports.declare", nil, nil)
	var f *fail.Error
	if !errors.As(err, &f) || f.Type != fail.FarTooOld || f.Detail["version"] != "v3.2.0" || f.Subject != "linux-1" {
		t.Errorf("asking linux-1 on v3.2.0 for ports.declare: %v, want far-too-old naming linux-1 and v3.2.0", err)
	}
	if len(e.lines) != 2 || e.lines[0] != unix.Guest || e.lines[1] != unix.Guest {
		t.Errorf("the engine ran %q inside the VM, want only %q", e.lines, unix.Guest)
	}
}

func TestAReplyToAnAgentTravelsOnTheGuestsStdinNeverOnItsCommandLine(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v3.2.0")}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{} }, nil).Dial(t.Context(), "jterrazz-macbook-pro")
	text := "yes, merge it; then `rm -rf build` && say $HOME"
	in, _ := json.Marshal(map[string]string{"Session": "s-1", "Text": text})
	_, _ = ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "agents.reply", in, nil)
	if len(e.lines) != 2 || e.lines[0] != unix.Guest || e.lines[1] != unix.Guest {
		t.Fatalf("the engine ran %q inside the VM, want only %q", e.lines, unix.Guest)
	}
	sent, _ := json.Marshal(text)
	if !strings.Contains(e.stdin[1], strings.Trim(string(sent), `"`)) {
		t.Errorf("the reply is not in the guest's stdin %q", e.stdin[1])
	}
}

func TestAGuestOfAReleaseBeforeTheFloorIsRefusedBeforeTheRequestReachesIt(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v3.1.0")}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{} }, imageOf).Dial(t.Context(), "jterrazz-macbook-pro")
	in, _ := json.Marshal(contracttest.InstallIn{App: "jq"})
	_, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", in, nil)
	var f *fail.Error
	if !errors.As(err, &f) || f.Type != fail.FarTooOld || f.Subject != "linux-1" || f.Detail["version"] != "v3.1.0" || f.Detail["floor"] != api.Floor || f.Detail["image"] != "station-linux" {
		t.Errorf("installing jq on linux-1 at v3.1.0: %v, want far-too-old naming linux-1, v3.1.0, the floor and the image it was made from", err)
	}
	if len(e.stdin) != 1 || strings.Contains(e.stdin[0], "apps.install") {
		t.Errorf("the guest at v3.1.0 was sent %q, want its hello alone", e.stdin)
	}
}

func imageOf(vm string) string {
	if vm == "linux-1" {
		return "station-linux"
	}
	return ""
}

func TestAGuestWhoseBinaryTurnsOlderThanTheFloorBetweenTwoExecsIsRefusedAtTheSecond(t *testing.T) {
	e := &engine{vm: serve.Local(contracttest.Far(), contracttest.Mini, "v3.3.0"), then: serve.Local(contracttest.Far(), contracttest.Mini, "v3.1.0")}
	ep, _ := unix.New(e, func() contract.Hello { return contract.Hello{} }, imageOf).Dial(t.Context(), "jterrazz-macbook-pro")
	in, _ := json.Marshal(contracttest.InstallIn{App: "jq"})
	_, err := ep.Invoke(t.Context(), machine.Ref{Name: "linux-1"}, "apps.install", in, nil)
	var f *fail.Error
	if !errors.As(err, &f) || f.Type != fail.FarTooOld || f.Detail["version"] != "v3.1.0" || f.Detail["floor"] != api.Floor || f.Detail["image"] != "station-linux" {
		t.Errorf("installing jq on linux-1 whose station became v3.1.0 after its first hello: %v, want far-too-old naming v3.1.0, the floor and its image", err)
	}
}
