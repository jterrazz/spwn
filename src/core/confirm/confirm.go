package confirm

import (
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
)

type Action string

const (
	Install     Action = "install"
	Share       Action = "share"
	Start       Action = "start"
	Restart     Action = "restart"
	Edit        Action = "edit"
	Remove      Action = "remove"
	TakeBack    Action = "take-back"
	StopSharing Action = "stop-sharing"
	Undeclare   Action = "undeclare"
	OffHere     Action = "off-here"
	Overwrite   Action = "overwrite"
	Forget      Action = "forget"
	DeleteImage Action = "delete-image"
	RestartMac  Action = "restart-mac"
	StopEngine  Action = "stop-engine"
	Stop        Action = "stop"
	StopAgent   Action = "stop-agent"
	Delete      Action = "delete"
	Erase       Action = "erase"
	// value: build the image a create needs first, when this Mac has none.
	BuildImage Action = "build-image"
	// value: delete the throwaways whose agent finished, in one question.
	Cleanup Action = "cleanup"
)

type Level string

const (
	None     Level = "none"
	YesNo    Level = "yes-no"
	TypeName Level = "type-name"
)

const Yes = "yes"

const (
	Needed    fail.Code = "confirm.needed"
	Mismatch  fail.Code = "confirm.mismatch"
	YesNeeded fail.Code = "confirm.yes-needed"
	Declined  fail.Code = "confirm.declined"
)

func For(a Action, k machine.Kind) Level {
	// invariant: SPEC §7.5, the graded confirmations of every surface. An
	// action of no row asks yes or no.
	switch a {
	case Install, Share, Start, Restart, Edit:
		return None
	case Delete, Erase:
		return TypeName
	case Stop:
		if k == machine.Server {
			return TypeName
		}
		return YesNo
	default:
		return YesNo
	}
}

// value: what a gesture's question depends on beyond its machine's kind,
// as the page that asks it knows: a workspace thrown away when done, a
// server that answers its checks, a Mac that hosts a server.
type Fact string

const (
	Throwaway   Fact = "throwaway"
	Healthy     Fact = "healthy"
	HostsServer Fact = "hosts-server"
)

func Of(a Action, k machine.Kind, facts ...Fact) Level {
	// invariant: SPEC A5: the menu bar's alert asks at the level its facts
	// grade: a throwaway's delete yes or no, a healthy server's restart yes
	// or no (an outage), a Mac hosting a server its name typed to restart.
	// With no fact, the level is the call's own, For's.
	has := func(f Fact) bool { return slices.Contains(facts, f) }
	switch {
	case a == Delete && k == machine.Workspace && has(Throwaway):
		return YesNo
	case a == Restart && k == machine.Server && has(Healthy):
		return YesNo
	case a == RestartMac && has(HostsServer):
		return TypeName
	}
	return For(a, k)
}

func Require(l Level, name, given string) error {
	// invariant: given is what the caller answered: Yes for a yes-or-no
	// question, the typed name for a typed one. Yes never answers a typed one.
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
		if given == Yes {
			return nil
		}
		return fail.New(fail.NeedsConfirmation, YesNeeded, name, detail)
	}
}

func Ask(a Action, k machine.Kind, name, given string) error {
	// value: Require for the gesture a on a machine of kind k, its refusal
	// naming both, so its words can say what was not done.
	err := Require(For(a, k), name, given)
	var e *fail.Error
	if errors.As(err, &e) {
		e.Detail["action"], e.Detail["kind"] = string(a), string(k)
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
