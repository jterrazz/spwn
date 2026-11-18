package accounts

import "errors"

var (
	ErrNoVault = errors.New("kept secrets live only in a Mac's keychain")
	// value: the vault keeps nothing under the name asked.
	ErrNotKept = errors.New("not kept")
)

type VaultError struct {
	Name  string
	Cause error
}

func (e *VaultError) Error() string { return "keychain account " + e.Name + ": " + e.Cause.Error() }

func (e *VaultError) Unwrap() error { return e.Cause }
