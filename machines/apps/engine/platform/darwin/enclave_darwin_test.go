//go:build darwin && cgo

package darwin

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestTheEnclaveWrapsAKeyThatOnlyItsBlobOpens(t *testing.T) {
	blob, err := EnclaveCreate(AccessPresent)
	var none *NoEnclaveError
	if errors.As(err, &none) {
		t.Skip("this Mac has no Secure Enclave")
	}
	if err != nil {
		t.Fatal(err)
	}
	first, err := EnclavePublic(blob)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 65 || first[0] != 4 {
		t.Fatalf("the public half is %d bytes, first %#x; want an uncompressed P-256 point", len(first), first[0])
	}
	again, err := EnclavePublic(blob)
	if err != nil || !bytes.Equal(first, again) {
		t.Fatalf("a second read of the blob gave another key: %v", err)
	}
	for name, b := range map[string][]byte{"half the blob": blob[:len(blob)/2], "bytes no Enclave wrapped": []byte("not a key")} {
		if pub, pubErr := EnclavePublic(b); pubErr == nil || errors.As(pubErr, &none) {
			t.Errorf("%s: EnclavePublic = %x, %v; want a refusal", name, pub, pubErr)
		}
		if sig, signErr := EnclaveSign(b, AccessPresent, "a test", time.Second, []byte("m")); signErr == nil || errors.As(signErr, &none) {
			t.Errorf("%s: EnclaveSign = %x, %v; want a refusal", name, sig, signErr)
		}
	}
}

func TestAnUnattendedKeySignsAfterTheFirstUnlockWithNobodyPresentAndAnyOtherOnlyForAPresentOwner(t *testing.T) {
	const privateKeyUsage, userPresence = 1 << 30, 1 << 0
	for access, want := range map[Access]struct {
		class string
		flags uint64
	}{
		AccessUnattended: {class: "cku", flags: privateKeyUsage},
		AccessPresent:    {class: "aku", flags: privateKeyUsage | userPresence},
	} {
		class, flags, err := enclaveAccess(access)
		if err != nil || class != want.class || flags != want.flags {
			t.Errorf("access %d: class %q flags %#x, %v; want %q %#x", access, class, flags, err, want.class, want.flags)
		}
	}
}
