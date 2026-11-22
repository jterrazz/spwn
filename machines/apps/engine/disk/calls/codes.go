package calls

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

const (
	Unknown        fail.Code = "disk.unknown"
	Elsewhere      fail.Code = "disk.lives-elsewhere"
	NothingToFree  fail.Code = "disk.nothing-to-free"
	NeverTouched   fail.Code = "disk.never-touched"
	NothingToErase fail.Code = "disk.nothing-to-erase"
	AppRunning     fail.Code = "disk.app-running"
	NotMisplaced   fail.Code = "disk.not-misplaced"
	// value: a move's destination already holds what is not a cache of
	// its app's, so nothing moves.
	DestinationHeld fail.Code = "disk.destination-held"
	// value: a name free-unused was asked that is not among what nothing
	// uses: a machine is made from it or names it, it runs, it is gone, or
	// station did not make it.
	NotFreeable fail.Code = "disk.not-freeable"
	// value: free-unused was asked no name: it frees only what it names.
	NamesNeeded fail.Code = "disk.names-needed"
	// value: a machine record cannot be read, so no disk is known unused.
	RecordUnreadable fail.Code = "disk.record-unreadable"
)

func coded(err error) error {
	// invariant: the refusals of the Disk calls travel as their codes, worded
	// by the binary that prints them.
	var unknown *disk.UnknownError
	var elsewhere *disk.ElsewhereError
	var nothingToFree *disk.NothingToFreeError
	var never *disk.NeverTouchedError
	var nothingToErase *disk.NothingToEraseError
	var running *disk.RunningError
	var placed *disk.NotMisplacedError
	var unread *disk.UnreadRecordError
	var held *disk.DestinationHeldError
	switch {
	case errors.As(err, &unknown):
		return fail.New(fail.NotFound, Unknown, unknown.Name, nil)
	case errors.As(err, &elsewhere):
		return fail.New(fail.NotForThisMachine, Elsewhere, elsewhere.Name, map[string]string{"why": elsewhere.Error()})
	case errors.As(err, &nothingToFree):
		return fail.New(fail.Refused, NothingToFree, nothingToFree.Name, nil)
	case errors.As(err, &never):
		return fail.New(fail.Refused, NeverTouched, never.Name, map[string]string{"path": never.Path})
	case errors.As(err, &nothingToErase):
		return fail.New(fail.Refused, NothingToErase, nothingToErase.Name, map[string]string{"hint": nothingToErase.Hint})
	case errors.As(err, &running):
		return fail.New(fail.Failed, AppRunning, running.Name, map[string]string{"process": running.Process, "verb": running.Verb, "subject": running.Subject})
	case errors.As(err, &placed):
		return fail.New(fail.Refused, NotMisplaced, placed.Path, nil)
	case errors.As(err, &held):
		return fail.New(fail.Refused, DestinationHeld, held.Folder, map[string]string{"path": held.Path})
	case errors.As(err, &unread):
		return fail.New(fail.Failed, RecordUnreadable, unread.Path, nil)
	}
	return err
}
