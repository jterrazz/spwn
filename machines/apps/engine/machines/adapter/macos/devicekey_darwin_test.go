//go:build darwin && cgo

package macos

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func TestTheDeviceKeyIsMadeOnceAndKeptInAFileOnlyItsOwnerReads(t *testing.T) {
	files, state := stateDir(t)
	first, err := NewDeviceKey(files, state, time.Second).Public(t.Context())
	var none *contract.NoDeviceKeyError
	if errors.As(err, &none) {
		t.Skip("this Mac has no Secure Enclave")
	}
	if err != nil {
		t.Fatal(err)
	}
	file := DeviceKeyBlob(state)
	info, err := files.Stat(file)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("the device key's blob: %v %v; want mode 0600", info, err)
	}
	blob, err := files.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewDeviceKey(files, state, time.Second).Public(t.Context())
	if err != nil || !first.Equal(again) {
		t.Fatalf("a second process read another key: %v", err)
	}
	wrong := blob[:len(blob)/2]
	if writeErr := files.WriteFile(file, wrong, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if _, pubErr := NewDeviceKey(files, state, time.Second).Public(t.Context()); pubErr == nil {
		t.Fatal("a blob the Enclave did not wrap opened a key")
	}
	if kept, readErr := files.ReadFile(file); readErr != nil || !bytes.Equal(kept, wrong) {
		t.Fatalf("the refused blob was replaced: %v", readErr)
	}
}
