package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/src/ports"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	model "github.com/jterrazz/studio-station/src/machines"
)

func engineFailure(err error) (*fail.Error, bool) {
	var running *enginecalls.RunningError
	var macOnly *enginecalls.MacOnlyError
	var down *enginecalls.DownError
	var guestMacOnly *model.GuestOnMacError
	if f, ok := controlFailure(err); ok {
		return f, true
	}
	switch {
	case errors.As(err, &running):
		return usage(running.Error(), "station overview shows it"), true
	case errors.As(err, &macOnly):
		return usage(macOnly.Error(), "run it on a Mac: station engine start installs it"), true
	case errors.As(err, &down):
		return failed(down.Error(), "station engine start", err), true
	case errors.As(err, &guestMacOnly):
		return usage(guestMacOnly.Error(), "its Mac's engine runs it inside each VM"), true
	}
	return portsFailure(err)
}

func portsFailure(err error) (*fail.Error, bool) {
	var full *ports.RangeFull
	var perVM *ports.VMFull
	var badPort *ports.BadPortError
	var ignored *ports.IgnoredPortError
	var away *ports.MacAwayError
	var unknown *ports.UnknownVMError
	switch {
	case errors.As(err, &full):
		return failed(full.Error(), "station ports stop-sharing <vm> <port>", err), true
	case errors.As(err, &perVM):
		return failed(perVM.Error(), "station ports stop-sharing "+perVM.VM+" <port>", err), true
	case errors.As(err, &badPort):
		return usage(badPort.Error(), "a port is a number from 1 to 65535"), true
	case errors.As(err, &ignored):
		return usage(ignored.Error(), "station never shares port 22 nor a port below 1024"), true
	case errors.As(err, &away):
		return failed(away.Error(), away.Hint(), err), true
	case errors.As(err, &unknown):
		return usage(unknown.Error(), "station machines"), true
	}
	return nil, false
}

func controlFailure(err error) (*fail.Error, bool) {
	var notInstalled *enginecalls.NotInstalledError
	var stuck *enginecalls.StuckError
	var boot *enginecalls.BootError
	switch {
	case errors.As(err, &notInstalled):
		return failed(notInstalled.Error(), "station engine start", err), true
	case errors.As(err, &stuck):
		return failed(stuck.Error(), "station machines delete "+stuck.VM, err), true
	case errors.As(err, &boot):
		return failed(boot.Error(), boot.See, err), true
	}
	return nil, false
}

func unanswered(err error) bool {
	var e *fail.Error
	return errors.As(err, &e) && (e.Code == call.Unreachable || e.Code == call.VMMacUnreachable)
}

// invariant: a read of a VM that is not running is a finding, the VM named
// with the verb that starts it; an action there is refused.
func vmStopped(err error) bool {
	var e *fail.Error
	var op *enginecalls.OpError
	return errors.As(err, &e) && e.Code == enginecalls.VMStopped || errors.As(err, &op) && op.Code == enginecalls.VMStopped
}
