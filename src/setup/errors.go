package setup

import "errors"

var (
	ErrNameOnVM = errors.New("a VM's name is given when it is created")
	// value: a Mac with no name yet, set up where nobody can be asked it.
	ErrNeedsName = errors.New("this Mac needs a name: init asks it on a terminal")
)

type NameMismatchError struct{ Named, Asked string }

func (e *NameMismatchError) Error() string {
	return "this Mac is named \"" + e.Named + "\", not \"" + e.Asked + "\""
}

type BadNameError struct{ Name string }

func (e *BadNameError) Error() string { return "\"" + e.Name + "\" is not a machine name" }

type StepFailedError struct {
	Step  string
	Cause error
}

func (e *StepFailedError) Error() string { return "init step " + e.Step + ": " + e.Cause.Error() }
func (e *StepFailedError) Unwrap() error { return e.Cause }

type DifferentContentError struct{ Path string }

func (e *DifferentContentError) Error() string {
	return e.Path + " already exists with different content"
}

type BusyError struct{ Process string }

func (e *BusyError) Error() string { return e.Process + " is running: its files cannot move under it" }

type DestinationExistsError struct{ Path string }

func (e *DestinationExistsError) Error() string { return e.Path + " already exists" }

type CrossDeviceError struct{ From, To string }

func (e *CrossDeviceError) Error() string {
	return e.From + " and " + e.To + " are on different volumes: station moves, never copies"
}
