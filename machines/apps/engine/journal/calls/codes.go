package calls

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

const (
	UnknownMac     fail.Code = "journal.unknown-mac"
	UnknownMachine fail.Code = "journal.unknown-machine"
	BadDay         fail.Code = "journal.bad-day"
	Unreadable     fail.Code = "journal.unreadable"
)

func coded(err error) error {
	// invariant: a journal that does not read travels as its code, its file
	// named, worded by the binary that prints it.
	var unread *store.UnreadableError
	if errors.As(err, &unread) {
		return fail.New(fail.Failed, Unreadable, unread.Path, map[string]string{"path": unread.Path})
	}
	return err
}

func Applies(p machine.Profile) machine.Verdict {
	// invariant: a Mac keeps what was done on it and on its VMs, so its
	// journal is read on a Mac alone.
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " is a " + p.Kind.Word(p.System.OS) + ": its Mac keeps what was done on it")
	}
	return machine.Yes
}
