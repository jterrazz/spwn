package confirm

import (
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Action string

const (
	Install     Action = "install"
	Share       Action = "share"
	Start       Action = "start"
	Restart     Action = "restart"
	Edit        Action = "edit"
	Remove      Action = "remove"
	StopSharing Action = "stop-sharing"
	// value: a tool's sign-in on a machine ends; the token stays valid at its provider.
	SignOut Action = "sign-out"
	// value: a password leaves this Mac's vault, and the grants made from it end.
	RemovePassword Action = "remove-password"
	// value: a grant ends; the password stays in the vault.
	RemoveGrant   Action = "remove-grant"
	Undeclare     Action = "undeclare"
	OffHere       Action = "off-here"
	Overwrite     Action = "overwrite"
	Forget        Action = "forget"
	DeleteImage   Action = "delete-image"
	RestartMac    Action = "restart-mac"
	RestartServer Action = "restart-server"
	StopEngine    Action = "stop-engine"
	Stop          Action = "stop"
	Delete        Action = "delete"
	// value: an agent's Mind is deleted, every layer of it.
	DeleteAgent Action = "delete-agent"
	Erase       Action = "erase"
	// value: build the image a create needs first, when this Mac has none.
	BuildImage Action = "build-image"
	// value: delete the stopped ephemeral workspaces, in one question.
	Cleanup Action = "cleanup"
	// value: delete what nothing uses on a Mac: the image versions no
	// machine is made from, and the leftovers.
	FreeUnused Action = "free-unused"
	// value: move the apps' files out of the home root, to where the
	// settings init writes make each app look.
	Relocate Action = "relocate"
)

type Level string

const (
	None     Level = "none"
	YesNo    Level = "yes-no"
	TypeName Level = "type-name"
)

func (Level) Enum() []string { return []string{string(None), string(YesNo), string(TypeName)} }

type Consequence string

const (
	// value: what an act can lose, which decides what it asks (ADR-097): a
	// reversible act loses nothing, or removes what held nothing kept; one
	// that stops others ends what other machines or people use; one that
	// destroys deletes data nothing gives back.
	Reversible  Consequence = "reversible"
	StopsOthers Consequence = "stops-others"
	Destroys    Consequence = "destroys"
)

func (c Consequence) Level() Level {
	// invariant: ADR-097: a reversible act asks nothing and says its undo,
	// one that stops others asks one yes or no naming them, one that
	// destroys asks the name typed.
	switch c {
	case Reversible:
		return None
	case StopsOthers:
		return YesNo
	case Destroys:
	}
	return TypeName
}

const Yes = "yes"

const (
	Needed    fail.Code = "confirm.needed"
	Mismatch  fail.Code = "confirm.mismatch"
	YesNeeded fail.Code = "confirm.yes-needed"
	Declined  fail.Code = "confirm.declined"
)

type Fact string

const (
	// value: what is true of the machine a gesture acts on, beyond its
	// kind, that changes what the gesture can lose: an ephemeral VM, one
	// that runs (or whose run is unread), one someone holds, a server
	// already down, a Mac hosting a server.
	Ephemeral   Fact = "ephemeral"
	Running     Fact = "running"
	Held        Fact = "held"
	Down        Fact = "down"
	HostsServer Fact = "hosts-server"
)

func Actions() []Action {
	return []Action{Install, Share, Start, Restart, Edit, Remove, StopSharing, SignOut, RemovePassword, RemoveGrant, Undeclare, OffHere,
		Overwrite, Forget, DeleteImage, RestartMac, RestartServer, StopEngine, Stop, Delete, DeleteAgent, Erase, BuildImage, Cleanup, FreeUnused, Relocate}
}

func ConsequenceOf(a Action, k machine.Kind, facts ...Fact) Consequence {
	// invariant: ADR-097, the one table of every surface: the command line,
	// the window and the menu bar all ask what this grades. An action of
	// no row destroys, so a new one asks the most until it is graded.
	has := func(f Fact) bool { return slices.Contains(facts, f) }
	switch a {
	case Install, Share, Start, Restart, Edit, StopSharing, Undeclare, BuildImage, Relocate:
		return Reversible
	case Stop:
		if k == machine.Server {
			return StopsOthers
		}
		return Reversible
	case OffHere:
		if has(HostsServer) {
			return StopsOthers
		}
		return Reversible
	case RestartServer:
		if has(Down) {
			return Reversible
		}
		return StopsOthers
	case Delete:
		if has(Ephemeral) && !has(Running) && !has(Held) {
			return Reversible
		}
		return Destroys
	case Remove, SignOut, RemovePassword, RemoveGrant, Overwrite, Forget, DeleteImage, RestartMac, StopEngine, Cleanup, FreeUnused:
		return StopsOthers
	case Erase, DeleteAgent:
	}
	return Destroys
}

func Of(a Action, k machine.Kind, facts ...Fact) Level {
	// value: the level a gesture asks at, its consequence's, on a machine
	// of kind k whose facts are known; with no fact, a delete grades the
	// VM as one that runs.
	return ConsequenceOf(a, k, facts...).Level()
}

func Require(l Level, name, given string) error {
	// invariant: given is what the caller answered: Yes for a yes-or-no
	// question, the typed name for a typed one. Yes never answers a typed
	// one; the name, naming what the act affects, answers both.
	detail := map[string]string{"level": string(l), "name": name, "typed": given}
	switch l {
	case None:
		return nil
	case TypeName:
		if given == name {
			return nil
		}
		if given == "" || given == Yes {
			return fail.New(fail.NeedsConfirmation, Needed, name, detail)
		}
		return fail.New(fail.NeedsConfirmation, Mismatch, name, detail)
	default:
		if given == Yes || given == name && name != "" {
			return nil
		}
		return fail.New(fail.NeedsConfirmation, YesNeeded, name, detail)
	}
}

func Ask(a Action, k machine.Kind, os machine.OS, name, given string, facts ...Fact) error {
	// value: Require for the gesture a on a machine of kind k whose system
	// is os and whose facts are known, its refusal naming all three, so its
	// words can say what was not done and call the machine by its word (a
	// Linux computer is no Mac).
	err := Require(Of(a, k, facts...), name, given)
	var e *fail.Error
	if errors.As(err, &e) {
		e.Detail["action"], e.Detail["kind"] = string(a), string(k)
		if os != "" {
			e.Detail["os"] = string(os)
		}
		if given == Yes {
			e.Detail["by"] = "yes"
		}
	}
	return err
}

func Answer(err error) (string, bool) {
	// value: the answer a confirmation the use case asked takes, once its
	// question is answered yes: the name for a typed question, Yes otherwise.
	var e *fail.Error
	if !errors.As(err, &e) || e.Type != fail.NeedsConfirmation || e.Code == Mismatch {
		return "", false
	}
	if Level(e.Detail["level"]) == TypeName {
		return e.Subject, true
	}
	return Yes, true
}

func (a Action) Consequence() string {
	// value: what the gesture does beyond its verb, the line under its
	// confirmation's title; a %s is the subject the question names.
	switch a {
	case Install:
		return "%s is installed on this machine."
	case Share:
		return "%s is reached from your other machines."
	case Start:
		return "%s starts, and its ports are shared again."
	case Restart:
		return "%s stops, then starts again: what it serves stops while it restarts."
	case Edit:
		return "%s changes on this machine."
	case Remove:
		return "%s leaves this machine; its data stays."
	case StopSharing:
		return "%s is no longer shared, and its reserved port is released."
	case SignOut:
		return "%s is signed out on this machine; its token stays valid at its provider."
	case RemovePassword:
		return "%s leaves this Mac's vault, and every grant made from it ends."
	case RemoveGrant:
		return "The grant of %s ends now; the password stays in this Mac's vault."
	case Undeclare:
		return "%s is no longer declared, so a listener on it is flagged again."
	case OffHere:
		return "Machines stops applying %s on this machine; its value now stays."
	case Overwrite:
		return "Your value of %s is replaced by the one the file asks."
	case Forget:
		return "%s leaves your machines; nothing on it changes."
	case DeleteImage:
		return "%s is deleted; the next create that needs it builds it again."
	case RestartMac:
		return "%s restarts: its VMs stop, and its servers start again once it is back."
	case RestartServer:
		return "It is a server: what it serves stops while it restarts."
	case StopEngine:
		return "Every VM of this Mac stops, and nothing is shared until the engine starts again."
	case Stop:
		return "%s stops: what it serves stops until you start it again."
	case Delete:
		return "%s and everything on its disk are deleted for good."
	case DeleteAgent:
		return "%s's Mind, every layer of it, is deleted for good."
	case Erase:
		return "The data of %s is deleted for good."
	case BuildImage:
		return "%s is built first, which takes several minutes."
	case Cleanup:
		return "Every stopped ephemeral workspace is deleted."
	case FreeUnused:
		return "Deleted for good: %s. No machine and no file of yours is touched."
	case Relocate:
		return "Moved where each app now looks, one rename each: %s."
	}
	return ""
}
