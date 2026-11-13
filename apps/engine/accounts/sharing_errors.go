package accounts

import (
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type NotAVMError struct{ Name string }

func (e *NotAVMError) Error() string {
	return e.Name + " is a Mac: accounts are shared with VMs only"
}

type OffMacError struct{ ThisMachine catalog.MachineKind }

func (e *OffMacError) Error() string {
	return "accounts are shared from a Mac; here is " + e.ThisMachine.Article()
}

type TermError struct {
	Account, VM string
	Term        time.Duration
}

func (e *TermError) Error() string {
	return "a share lasts 1 min to 7 d, not " + e.Term.String()
}

type NoStudioError struct{ VM string }

func (e *NoStudioError) Error() string { return "no studio command in \"" + e.VM + "\"" }

type StepError struct {
	VM, Step string
	Cause    error
}

func (e *StepError) Error() string {
	msg := e.Step + " failed for \"" + e.VM + "\""
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

func (e *StepError) Unwrap() error { return e.Cause }

func VMStudio(args ...string) string {
	// value: studio's command line inside a VM, its credential store a file
	// there: import reads the secrets on stdin, delete takes their names.
	return strings.Join(append([]string{"STUDIO_CREDENTIALS_BACKEND=file", "studio", "credentials"}, args...), " ")
}
