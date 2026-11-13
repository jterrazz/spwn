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
	}
	return err
}
