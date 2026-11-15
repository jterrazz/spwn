package accounts

type UnknownError struct{ Account string }

func (e *UnknownError) Error() string { return "no account named \"" + e.Account + "\"" }

type NotShareableError struct{ Account string }

func (e *NotShareableError) Error() string { return e.Account + " is not on the share list" }

type NotSignedInError struct{ Account, Mac string }

func (e *NotSignedInError) Error() string { return Label(e.Account) + " is not signed in on " + e.Mac }

type UnreadableError struct {
	Account, Mac string
	Log          string
	Cause        error
}

func (e *UnreadableError) Error() string {
	return Label(e.Account) + " could not be read on " + e.Mac + ": " + e.Cause.Error()
}

func (e *UnreadableError) Unwrap() error { return e.Cause }

type UnknownVMError struct{ VM string }

func (e *UnknownVMError) Error() string { return "no machine named \"" + e.VM + "\"" }

type VMStoppedError struct{ VM string }

func (e *VMStoppedError) Error() string { return e.VM + " is stopped" }

type ToServerError struct{ VM string }

func (e *ToServerError) Error() string {
	return e.VM + " is a server: station shares no account with it"
}
