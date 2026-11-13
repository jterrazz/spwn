package fail

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
)

type Exit int

const (
	ExitOK          Exit = 0
	ExitFailure     Exit = 1
	ExitUsage       Exit = 2
	ExitInterrupted Exit = 130
)

type Type string

const (
	Refused           Type = "refused"
	NotFound          Type = "not-found"
	NotForThisMachine Type = "not-for-this-machine"
	NeedsConfirmation Type = "needs-confirmation"
	NeedsTerminal     Type = "needs-terminal"
	Declined          Type = "declined"
	Unreachable       Type = "unreachable"
	EngineStopped     Type = "engine-stopped"
	FarTooOld         Type = "far-too-old"
	Failed            Type = "failed"
	Interrupted       Type = "interrupted"
)

func (t Type) Exit() Exit {
	// invariant: SPEC §12.2, the exit code computed once from the type.
	switch t {
	case Refused, NotFound, NotForThisMachine, NeedsConfirmation, NeedsTerminal:
		return ExitUsage
	case Interrupted:
		return ExitInterrupted
	default:
		return ExitFailure
	}
}

type Code string

type Error struct {
	// invariant: an error carries its Type and Code (v3), or its Message, Hint
	// and Exit (the errors kit.Translate words today); Type, when set, decides
	// the exit code.
	Type    Type
	Code    Code
	Subject string
	Detail  map[string]string
	Fix     finding.Fix
	Message string
	Hint    string
	Exit    Exit
	Cause   error
}

func New(t Type, code Code, subject string, detail map[string]string) *Error {
	return &Error{Type: t, Code: code, Subject: subject, Detail: detail}
}

func (e *Error) Error() string {
	switch {
	case e.Message != "":
		return e.Message
	case e.Cause != nil:
		return e.Cause.Error()
	case e.Code != "" && e.Subject != "":
		return string(e.Code) + ": " + e.Subject
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.Cause }

func Silent(code Exit) *Error {
	return &Error{Exit: code}
}

func Interruption() *Error {
	return &Error{Exit: ExitInterrupted, Cause: context.Canceled}
}

func ExitOf(err error) Exit {
	if err == nil {
		return ExitOK
	}
	var e *Error
	if errors.As(err, &e) {
		if e.Type != "" {
			return e.Type.Exit()
		}
		return e.Exit
	}
	if errors.Is(err, context.Canceled) {
		return ExitInterrupted
	}
	return ExitFailure
}

type Said struct{ Message, Hint string }

func (s *Said) Error() string { return s.Message }
