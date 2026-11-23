//go:build !darwin || !cgo

package darwin

import "time"

func EnclaveCreate(Access) ([]byte, error) {
	return nil, &NoEnclaveError{Why: "no Secure Enclave on this system"}
}

func EnclavePublic([]byte) ([]byte, error) {
	return nil, &NoEnclaveError{Why: "no Secure Enclave on this system"}
}

func EnclaveSign([]byte, Access, string, time.Duration, []byte) ([]byte, error) {
	return nil, &NoEnclaveError{Why: "no Secure Enclave on this system"}
}
