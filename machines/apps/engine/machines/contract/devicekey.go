package contract

import (
	"context"
	"crypto/ecdsa"
)

type DeviceKey interface {
	// invariant: one key per device, made on its first ask and kept where it
	// was made: its public half and its signatures leave the device, the
	// private half never does.
	Public(ctx context.Context) (*ecdsa.PublicKey, error)
	// value: an ASN.1 ECDSA signature over the SHA-256 digest of message.
	Sign(ctx context.Context, message []byte) ([]byte, error)
}

type NoDeviceKeyError struct {
	// value: why the machine has nowhere to keep a key that cannot leave it.
	Why string
}

func (e *NoDeviceKeyError) Error() string { return "this machine cannot keep a device key: " + e.Why }
