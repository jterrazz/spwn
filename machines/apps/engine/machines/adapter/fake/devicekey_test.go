package fake

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
)

func TestDeviceKeyContract(t *testing.T) {
	contracttest.DeviceKey(t, func(*testing.T) contract.DeviceKey { return &DeviceKey{} })
}

func TestAnAbsentDeviceKeyRefusesEveryAsk(t *testing.T) {
	k := &DeviceKey{Absent: true}
	var none *contract.NoDeviceKeyError
	if _, err := k.Public(t.Context()); !errors.As(err, &none) {
		t.Fatalf("Public = %v; want a NoDeviceKeyError", err)
	}
	if _, err := k.Sign(t.Context(), make([]byte, 32)); !errors.As(err, &none) {
		t.Fatalf("Sign = %v; want a NoDeviceKeyError", err)
	}
}
