package darwin

import (
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestASystemWithNoEnclaveKeepsNoKey(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("a Mac has its Secure Enclave")
	}
	var none *NoEnclaveError
	if _, err := EnclavePublic("k"); !errors.As(err, &none) {
		t.Fatalf("EnclavePublic = %v; want a NoEnclaveError", err)
	}
	if _, err := EnclaveSign("k", "r", time.Second, make([]byte, 32)); !errors.As(err, &none) {
		t.Fatalf("EnclaveSign = %v; want a NoEnclaveError", err)
	}
}
