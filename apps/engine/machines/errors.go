package machines

import (
	"errors"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type RunningUnderAgentError struct{ Names []string }

func (e *RunningUnderAgentError) Error() string {
	return strings.Join(e.Names, ", ") + " still runs under its round-3 agent"
}

type EngineDownError struct {
	Mac       string
	Installed bool
}

func (e *EngineDownError) Error() string {
	if e.Installed {
		return "the engine is stopped on " + e.Mac
	}
	return "the engine is not installed on " + e.Mac
}

type BadNetworkError struct {
	Value  string
	Bridge bool
}

func (e *BadNetworkError) Error() string {
	if e.Bridge {
		return "--network bridged is for macOS images: a linux workspace stays behind its Mac"
	}
	return "unknown network \"" + e.Value + "\": nat or bridged"
}

type KindRuleError struct {
	// value: a create a VM's kind refuses: Why says the rule, Fix the way out.
	Why, Fix string
}

func (e *KindRuleError) Error() string { return e.Why }

type BadCheckError struct{ Value string }

func (e *BadCheckError) Error() string {
	return "--check \"" + e.Value + "\" is neither a URL nor host:port"
}

type BadCreatorError struct{ Session string }

func (e *BadCreatorError) Error() string {
	return "STATION_SESSION \"" + e.Session + "\" is not a session's name"
}

type BadPurposeError struct{ Purpose string }

func (e *BadPurposeError) Error() string {
	return "a purpose is one line of at most " + strconv.Itoa(PurposeMax) + " characters"
}

type BadLifetimeError struct{ Value string }

func (e *BadLifetimeError) Error() string {
	return "unknown lifetime \"" + e.Value + "\": ephemeral or persistent"
}

type UnknownImageError struct{ Image string }

func (e *UnknownImageError) Error() string { return "unknown image \"" + e.Image + "\"" }

type NoImageError struct {
	Image, Mac string
	// value: station only fetches the image: it has no recipe to build it from.
	Fetched bool
}

func (e *NoImageError) Error() string { return "image " + e.Image + " is not built on " + e.Mac }

type NoEngineError struct {
	Hypervisor string
	Cause      error
}

func (e *NoEngineError) Error() string { return e.Hypervisor + " is not installed or does not answer" }

func (e *NoEngineError) Unwrap() error { return e.Cause }

type TooBigError struct {
	Mac             string
	AskedGB, LeftGB int
}

func (e *TooBigError) Error() string {
	return "not enough memory on " + e.Mac + ": " + strconv.Itoa(e.AskedGB) + " GB asked, " + strconv.Itoa(e.LeftGB) + " GB left for VMs"
}

type CreateFailedError struct {
	Name, Step string
	Cause      error
}

func (e *CreateFailedError) Error() string {
	text := "creating " + e.Name + " failed at " + e.Step + ", and was rolled back"
	if e.Cause != nil {
		text += ": " + strings.TrimSpace(e.Cause.Error())
	}
	return text
}

func (e *CreateFailedError) Unwrap() error { return e.Cause }

type BadSizeError struct{ Flag, Value string }

func (e *BadSizeError) Error() string { return "--" + e.Flag + " \"" + e.Value + "\" is not a size" }

type SizeNeedsUnitError struct{ Flag, Value string }

func (e *SizeNeedsUnitError) Error() string { return "\"" + e.Value + "\" needs a unit" }

type DidNotBootError struct {
	Name  string
	Cause error
}

func (e *DidNotBootError) Error() string {
	text := e.Name + " did not boot"
	if e.Cause != nil {
		text += ": " + strings.TrimSpace(e.Cause.Error())
	}
	return text
}

type UnhealthyError struct{ Name, Step, Detail string }

func (e *UnhealthyError) Error() string {
	return e.Name + " is kept, failing " + e.Step + ": " + e.Detail
}

var ErrEngineDown = errors.New("the engine is not running")

type NothingToChangeError struct{ Name string }

func (e *NothingToChangeError) Error() string { return "say what to change on " + e.Name }

type DiskShrinksError struct {
	Name string
	Has  int64
}

func (e *DiskShrinksError) Error() string { return "a VM's disk only grows" }

type UnknownMachineError struct{ Name string }

func (e *UnknownMachineError) Error() string { return "no machine named \"" + e.Name + "\"" }

type ThisMachineError struct{ Name, Verb string }

func (e *ThisMachineError) Error() string {
	if e.Verb == VerbForget {
		return "this is This Mac: station forgets only the other Macs"
	}
	return "\"" + e.Name + "\" is this machine"
}

type NotAMacError struct{ Name, Verb string }

func (e *NotAMacError) Error() string {
	if e.Verb == VerbForget {
		return e.Name + " is a VM: a VM is deleted, never forgotten"
	}
	return "\"" + e.Name + "\" is a VM, not a Mac"
}

type OnlyOnMacError struct{ ThisMachine catalog.MachineKind }

func (e *OnlyOnMacError) Error() string {
	return "only a Mac does this; here is " + e.ThisMachine.Article()
}

type NameTakenError struct {
	Name string
	VM   bool
	Mac  bool
	// value: the Mac a VM of that name is on, by the Name machines.list
	// gives it; for an agent, the workspace it runs on.
	On string
	// value: the agent that holds the name, its own or the name's ·<n>.
	Agent string
}

func (e *NameTakenError) Error() string {
	switch {
	case e.Agent == e.Name:
		return "\"" + e.Name + "\" is taken (an agent on " + e.On + ")"
	case e.Agent != "":
		return "\"" + e.Name + "\" is taken (agent " + e.Agent + " on " + e.On + ")"
	}
	if e.Mac {
		return "\"" + e.Name + "\" already names a Mac"
	}
	if e.VM {
		return "\"" + e.Name + "\" is taken (on " + e.On + ")"
	}
	return "machine \"" + e.Name + "\" already exists"
}

type BadNameError struct{ Name string }

func (e *BadNameError) Error() string { return "\"" + e.Name + "\" is not a valid name" }

type NoHardwareAddressError struct{ Name string }

func (e *NoHardwareAddressError) Error() string {
	return e.Name + " never said its hardware address: it answers once first"
}

type UnreachableError struct{ Name string }

func (e *UnreachableError) Error() string { return e.Name + " does not answer" }

type RestartTimeoutError struct{ Name, Phase string }

func (e *RestartTimeoutError) Error() string {
	if e.Phase == PhaseStop {
		return e.Name + " did not restart"
	}
	return e.Name + " did not come back"
}

type RestartRefusedError struct{ Name, Detail string }

func (e *RestartRefusedError) Error() string {
	return "\"" + e.Name + "\" refused to restart: " + e.Detail
}

type NeedsTerminalError struct{ Verb string }

func (e *NeedsTerminalError) Error() string { return e.Verb + " needs a terminal" }

type SSHConfigError struct {
	Path       string
	Cause      error
	RolledBack bool
}

func (e *SSHConfigError) Error() string {
	text := "updating the ssh config failed"
	if e.RolledBack {
		text += " (registry rolled back)"
	}
	return text + ": " + e.Cause.Error()
}

func (e *SSHConfigError) Unwrap() error { return e.Cause }

type RecordError struct {
	Path  string
	Cause error
}

func (e *RecordError) Error() string {
	return "cannot read the VM record " + e.Path + ": " + e.Cause.Error()
}

func (e *RecordError) Unwrap() error { return e.Cause }

type NotAVMError struct{ Name, Verb string }

func (e *NotAVMError) Error() string { return e.Name + " is a Mac" }

type NotThisMacError struct{ Name, Mac string }

func (e *NotThisMacError) Error() string { return e.Name + " runs on " + e.Mac + ", not here" }

type StoppedError struct {
	Name     string
	Lifetime Lifetime
}

func (e *StoppedError) Error() string {
	return e.Name + " is stopped"
}

type SSHDClosedError struct {
	Name, Addr string
	Cause      error
}

func (e *SSHDClosedError) Error() string { return e.Name + "'s sshd does not answer on " + e.Addr }

func (e *SSHDClosedError) Unwrap() error { return e.Cause }

const (
	VerbForget  = "forget"
	PhaseStop   = "stop"
	PhaseReturn = "return"
)
