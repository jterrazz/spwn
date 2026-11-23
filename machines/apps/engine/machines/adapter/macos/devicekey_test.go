package macos

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func stateDir(t *testing.T) (files local.FS, state string) {
	t.Helper()
	return local.New(nil, nil, io.Discard, io.Discard).FS, t.TempDir()
}

func TestAMachineWithNoEnclaveRefusesTheDeviceKeyAndKeepsItsFile(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("a Mac has its Secure Enclave")
	}
	files, state := stateDir(t)
	key := NewDeviceKey(files, state, time.Second, darwin.AccessPresent)
	var none *contract.NoDeviceKeyError
	if _, err := key.Public(t.Context()); !errors.As(err, &none) {
		t.Fatalf("Public = %v; want a NoDeviceKeyError", err)
	}
	if _, err := files.Stat(DeviceKeyBlob(state, darwin.AccessPresent)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a machine with no enclave wrote a device key: %v", err)
	}
	if _, err := key.Sign(t.Context(), []byte("m")); err == nil || !strings.Contains(err.Error(), "device key is gone") {
		t.Fatalf("Sign with no device key = %v; want it gone", err)
	}
	if err := files.WriteFile(DeviceKeyBlob(state, darwin.AccessPresent), []byte("wrapped"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := key.Public(t.Context()); !errors.As(err, &none) {
		t.Fatalf("Public over a blob = %v; want a NoDeviceKeyError", err)
	}
	if blob, err := files.ReadFile(DeviceKeyBlob(state, darwin.AccessPresent)); err != nil || string(blob) != "wrapped" {
		t.Fatalf("the refused blob was replaced: %q %v", blob, err)
	}
}

// value: an Enclave in memory: each blob it makes opens its own key, and it
// keeps the access every signature was asked with.
type memoryEnclave struct {
	keys   map[string]*ecdsa.PrivateKey
	signed []darwin.Access
}

func (e *memoryEnclave) create(access darwin.Access) ([]byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	blob := fmt.Sprintf("wrapped %d for %d", len(e.keys), access)
	e.keys[blob] = key
	return []byte(blob), nil
}

func (e *memoryEnclave) public(blob []byte) ([]byte, error) {
	key, ok := e.keys[string(blob)]
	if !ok {
		return nil, errors.New("not a blob of this Enclave")
	}
	return key.PublicKey.Bytes()
}

func (e *memoryEnclave) sign(blob []byte, access darwin.Access, _ string, _ time.Duration, message []byte) ([]byte, error) {
	key, ok := e.keys[string(blob)]
	if !ok {
		return nil, errors.New("not a blob of this Enclave")
	}
	e.signed = append(e.signed, access)
	digest := sha256.Sum256(message)
	return ecdsa.SignASN1(rand.Reader, key, digest[:])
}

func TestAMacRunAloneReplacesItsKeyByOneThatSignsWithNobodyPresentAndBackWhenSomeoneUsesItAgain(t *testing.T) {
	files, state := stateDir(t)
	enclave := &memoryEnclave{keys: map[string]*ecdsa.PrivateKey{}}
	started := func(access darwin.Access) *DeviceKey {
		k := NewDeviceKey(files, state, time.Second, access)
		k.enclave = enclave
		return k
	}
	public := func(k *DeviceKey) *ecdsa.PublicKey {
		t.Helper()
		pub, err := k.Public(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		return pub
	}
	kept := func(access darwin.Access, want bool) {
		t.Helper()
		info, err := files.Stat(DeviceKeyBlob(state, access))
		switch {
		case want && (err != nil || info.Mode().Perm() != 0o600):
			t.Fatalf("the key of access %d: %v %v; want it kept, mode 0600", access, info, err)
		case !want && !errors.Is(err, fs.ErrNotExist):
			t.Fatalf("the key of access %d is still kept: %v", access, err)
		}
	}
	used := public(started(darwin.AccessPresent))
	alone := started(darwin.AccessUnattended)
	if public(alone).Equal(used) {
		t.Fatal("a Mac run alone kept the key that asks for a present owner")
	}
	kept(darwin.AccessUnattended, true)
	kept(darwin.AccessPresent, false)
	if _, err := alone.Sign(t.Context(), []byte("m")); err != nil {
		t.Fatal(err)
	}
	if len(enclave.signed) != 1 || enclave.signed[0] != darwin.AccessUnattended {
		t.Fatalf("signed with %v; want one unattended signature", enclave.signed)
	}
	if !public(started(darwin.AccessUnattended)).Equal(public(alone)) {
		t.Fatal("a second start of a Mac run alone made another key")
	}
	back := public(started(darwin.AccessPresent))
	if back.Equal(public(alone)) || back.Equal(used) {
		t.Fatal("a Mac used again did not get a new key that asks for a present owner")
	}
	kept(darwin.AccessPresent, true)
	kept(darwin.AccessUnattended, false)
}
