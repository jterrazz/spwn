package disk

import (
	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type UnknownError struct{ Name string }

func (e *UnknownError) Error() string { return "no disk row for \"" + e.Name + "\"" }

type ElsewhereError struct {
	Name  string
	Kinds []catalog.MachineKind
}

func (e *ElsewhereError) Error() string { return e.Name + " is for " + apps.LivesOn(e.Kinds) }

type NothingToFreeError struct{ Name string }

func (e *NothingToFreeError) Error() string { return e.Name + " keeps nothing station can rebuild" }

type NeverTouchedError struct{ Name, Path string }

func (e *NeverTouchedError) Error() string {
	return e.Path + " is never touched: station erases no key, token or setting"
}

type NothingToEraseError struct{ Name, Hint string }

func (e *NothingToEraseError) Error() string { return e.Name + " has no data station can erase" }

type RunningError struct {
	// value: the app whose row it is, the process that runs, and what its
	// run forbids: "erase" or "move".
	Name, Process, Verb, Subject string
}

func (e *RunningError) Error() string { return e.Process + " is running" }

type NotMisplacedError struct{ Path string }

func (e *NotMisplacedError) Error() string { return e.Path + " is not misplaced" }
