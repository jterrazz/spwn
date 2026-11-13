package settings

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type UnknownItemError struct{ Name string }

func (e *UnknownItemError) Error() string { return "unknown setting \"" + e.Name + "\"" }

type NeedsRootError struct{ Item, Verb string }

func (e *NeedsRootError) Error() string { return e.Item + " needs root" }

type WrongLivesOnError struct {
	Item        string
	Kinds       []catalog.MachineKind
	ThisMachine catalog.MachineKind
}

func (e *WrongLivesOnError) Error() string {
	kinds := make([]string, len(e.Kinds))
	for i, p := range e.Kinds {
		kinds[i] = string(p)
	}
	return e.Item + " lives on " + strings.Join(kinds, ", ") + "; here is " + e.ThisMachine.Article()
}

type NotReversibleError struct{ Item string }

func (e *NotReversibleError) Error() string { return e.Item + " cannot be uninstalled by station" }

type NeedsTerminalError struct{ Item string }

func (e *NeedsTerminalError) Error() string { return e.Item + " needs a terminal" }

type UnhandledError struct {
	Item string
	// value: the builtin a newer catalog named, which this station does not
	// carry; empty for an effect it knows and no Applier writes.
	Builtin string
}

func (e *UnhandledError) Error() string {
	if e.Builtin != "" {
		return "this station does not know " + e.Builtin + ": update station"
	}
	return "no handler for setting \"" + e.Item + "\""
}
