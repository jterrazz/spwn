package darwin

type NoEnclaveError struct {
	// value: why the Mac cannot keep a Secure Enclave key, worded by the system.
	Why string
}

func (e *NoEnclaveError) Error() string { return e.Why }
