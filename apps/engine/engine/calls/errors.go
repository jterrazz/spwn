package calls

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

const (
	VMStopped        fail.Code = "vm_stopped"
	UnknownVM        fail.Code = "unknown_vm"
	UnknownOp        fail.Code = "unknown_op"
	UnknownPage      fail.Code = "unknown_page"
	GuestAbsent      fail.Code = "guest_absent"
	GuestTimeout     fail.Code = "guest_timeout"
	IgnoredPort      fail.Code = "ignored_port"
	NotAllowed       fail.Code = "not_allowed"
	RangeFull        fail.Code = "range_full"
	AwakeRun         fail.Code = "awake_run"
	AwakeRange       fail.Code = "awake_range"
	AwakeGuard       fail.Code = "awake_guard"
	AwakeUnsupported fail.Code = "awake_unsupported"
	Internal         fail.Code = "internal"
)

type NotInstalledError struct{}

func (*NotInstalledError) Error() string { return "the engine is not installed" }

type NoJobManagerError struct {
	// value: the job manager the engine runs under, and its command, as
	// the Mac's adapter names them.
	Manager, Command string
}

func (e *NoJobManagerError) Error() string {
	return "the engine needs " + e.Manager + ": " + e.Command + " is not found"
}

type StuckError struct{ VM string }

func (e *StuckError) Error() string { return e.VM + " did not stop; the engine keeps running" }

type BootError struct {
	Message string
	// value: the command line the owner types to read the engine's job.
	See   string
	Cause error
}

func (e *BootError) Error() string { return e.Message }

func (e *BootError) Unwrap() error { return e.Cause }

type DownError struct{}

func (*DownError) Error() string { return "the engine is not running" }

type RunningError struct{ PID int }

func (e *RunningError) Error() string {
	return "an engine already runs (pid " + strconv.Itoa(e.PID) + ")"
}

type MacOnlyError struct{ Kind catalog.MachineKind }

func (e *MacOnlyError) Error() string {
	return "the engine runs only on a Mac; this machine is " + e.Kind.Article()
}

type OpError struct {
	// value: an engine refusal worded where it is raised: its code travels
	// in the error frame, its words are the engine's.
	Code    fail.Code `json:"code"`
	Message string    `json:"message"`
	Hint    string    `json:"hint"`
}

func (e *OpError) Error() string { return e.Message }

func (e *OpError) Refused() bool {
	return strings.HasPrefix(string(e.Code), "unknown_") || e.Code == VMStopped
}
