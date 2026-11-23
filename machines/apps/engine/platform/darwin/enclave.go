package darwin

type NoEnclaveError struct {
	// value: why the Mac cannot keep a Secure Enclave key, worded by the system.
	Why string
}

func (e *NoEnclaveError) Error() string { return e.Why }

type Access int

// invariant: who a device key signs for is fixed by the Enclave when it makes
// the key: a key never changes its access, so another access is another key.
const (
	// value: an owner present at the unlocked Mac, asked once a session.
	AccessPresent Access = iota
	// value: nobody, from the Mac's first unlock after its start on, so a
	// Mac run alone signs with its screen locked.
	AccessUnattended
)
