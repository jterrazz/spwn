package calls

import (
	"cmp"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

// invariant: a machine's gesture keeps one line on the machine, its text
// what was done to it, or that it was not when the gesture failed; one
// with no machine named keeps none.
func gestureLine(vm, done string, err error) []journal.Line {
	if vm == "" {
		return nil
	}
	text := vm + " " + done
	if err != nil {
		text = vm + " not " + done
	}
	return []journal.Line{{Machine: vm, Text: text, Subject: &journal.Subject{Kind: journal.Machine, Name: vm}}}
}

func keptDelete(in DeleteIn, _ machines.DeleteReport, err error) []journal.Line {
	return gestureLine(in.VM, "deleted", err)
}

func keptRestart(in RestartIn, _ machines.RestartReport, err error) []journal.Line {
	return gestureLine(in.Name, "restarted", err)
}

// invariant: a start or a stop that found the VM as asked did nothing,
// and keeps nothing.
func keptStart(in VMIn, out StartOut, err error) []journal.Line {
	if out.Already {
		return nil
	}
	return gestureLine(in.VM, "started", err)
}

func keptStop(in StopIn, out StopOut, err error) []journal.Line {
	if out.Already {
		return nil
	}
	return gestureLine(in.VM, "stopped", err)
}

// invariant: a create is done by the session that asked it, when one did,
// and keeps the purpose it was asked for.
func keptCreate(name, purpose, creator string, err error) []journal.Line {
	lines := gestureLine(name, "created", err)
	for i := range lines {
		lines[i].Purpose, lines[i].By = purpose, creator
	}
	return lines
}

func keptWorkspace(in CreateIn, out machines.Created, err error) []journal.Line {
	return keptCreate(cmp.Or(out.VM.Name, in.Request.Name), in.Request.Purpose, in.Request.Creator, err)
}

func keptServer(in CreateServerIn, out machines.Created, err error) []journal.Line {
	return keptCreate(cmp.Or(out.VM.Name, in.Name), in.Purpose, in.Creator, err)
}
