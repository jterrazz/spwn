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
	if _, err := EnclaveCreate(AccessUnattended); !errors.As(err, &none) {
		t.Fatalf("EnclaveCreate = %v; want a NoEnclaveError", err)
	}
	if _, err := EnclavePublic([]byte("k")); !errors.As(err, &none) {
		t.Fatalf("EnclavePublic = %v; want a NoEnclaveError", err)
	}
	if _, err := EnclaveSign([]byte("k"), AccessUnattended, "r", time.Second, []byte("m")); !errors.As(err, &none) {
		t.Fatalf("EnclaveSign = %v; want a NoEnclaveError", err)
	}
}
