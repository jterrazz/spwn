//go:build !darwin || !cgo

package darwin

import "time"

func EnclavePublic(string) ([]byte, error) {
	return nil, &NoEnclaveError{Why: "no Secure Enclave on this system"}
}

func EnclaveSign(string, string, time.Duration, []byte) ([]byte, error) {
	return nil, &NoEnclaveError{Why: "no Secure Enclave on this system"}
}
