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

func (e *NothingToFreeError) Error() string { return e.Name + " keeps nothing Machines can rebuild" }

type NeverTouchedError struct{ Name, Path string }

func (e *NeverTouchedError) Error() string {
	return e.Path + " is never touched: Machines erases no key, token or setting"
}

type NothingToEraseError struct{ Name, Hint string }

func (e *NothingToEraseError) Error() string { return e.Name + " has no data Machines can erase" }

type RunningError struct {
	// value: the app whose row it is, the process that runs, and what its
	// run forbids: "erase" or "move".
	Name, Process, Verb, Subject string
}

func (e *RunningError) Error() string { return e.Process + " is running" }

type NotMisplacedError struct{ Path string }

func (e *NotMisplacedError) Error() string { return e.Path + " is not misplaced" }

type LinkedError struct{ Path, Target string }

func (e *LinkedError) Error() string { return e.Path + " is a link to " + e.Target + ", not moved" }

type UnreadRecordError struct{ Path string }

func (e *UnreadRecordError) Error() string {
	return e.Path + " cannot be read: nothing is freed until it can"
}

type DestinationHeldError struct{ Folder, Path string }

func (e *DestinationHeldError) Error() string { return e.Path + " already exists" }
