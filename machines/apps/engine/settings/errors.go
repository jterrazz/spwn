package settings

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type UnknownPackageError struct{ Name string }

func (e *UnknownPackageError) Error() string { return "unknown setting \"" + e.Name + "\"" }

type NeedsRootError struct{ Package, Verb string }

func (e *NeedsRootError) Error() string { return e.Package + " needs root" }

type WrongLivesOnError struct {
	Package     string
	Kinds       []catalog.MachineKind
	ThisMachine catalog.MachineKind
	// value: the package lives on Macs, and this one is a laptop, which
	// the Always-on Mac group does not reach.
	Laptop bool
}

func (e *WrongLivesOnError) Error() string {
	if e.Laptop {
		return e.Package + " is for an always-on Mac; this Mac is a laptop"
	}
	kinds := make([]string, len(e.Kinds))
	for i, p := range e.Kinds {
		kinds[i] = string(p)
	}
	return e.Package + " lives on " + strings.Join(kinds, ", ") + "; here is " + e.ThisMachine.Article()
}

type NotReversibleError struct{ Package string }

func (e *NotReversibleError) Error() string { return e.Package + " cannot be uninstalled by Machines" }

type NeedsTerminalError struct{ Package string }

func (e *NeedsTerminalError) Error() string { return e.Package + " needs a terminal" }

type UnhandledError struct {
	Package string
	// value: the builtin a newer catalog named, which this station does not
	// carry; empty for an effect it knows and no Applier writes.
	Builtin string
}

func (e *UnhandledError) Error() string {
	if e.Builtin != "" {
		return "this station does not know " + e.Builtin + ": update station"
	}
	return "no handler for setting \"" + e.Package + "\""
}
