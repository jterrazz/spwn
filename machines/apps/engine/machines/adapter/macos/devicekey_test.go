package macos

import (
	"errors"
	"io"
	"io/fs"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
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
	key := NewDeviceKey(files, state, time.Second)
	var none *contract.NoDeviceKeyError
	if _, err := key.Public(t.Context()); !errors.As(err, &none) {
		t.Fatalf("Public = %v; want a NoDeviceKeyError", err)
	}
	if _, err := files.Stat(DeviceKeyBlob(state)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a machine with no enclave wrote a device key: %v", err)
	}
	if _, err := key.Sign(t.Context(), []byte("m")); err == nil || !strings.Contains(err.Error(), "device key is gone") {
		t.Fatalf("Sign with no device key = %v; want it gone", err)
	}
	if err := files.WriteFile(DeviceKeyBlob(state), []byte("wrapped"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := key.Public(t.Context()); !errors.As(err, &none) {
		t.Fatalf("Public over a blob = %v; want a NoDeviceKeyError", err)
	}
	if blob, err := files.ReadFile(DeviceKeyBlob(state)); err != nil || string(blob) != "wrapped" {
		t.Fatalf("the refused blob was replaced: %q %v", blob, err)
	}
}
