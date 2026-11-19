package fake

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

type DeviceKey struct {
	// value: a device key held in memory, made on its first ask; Absent is a
	// machine that has nowhere to keep one.
	Absent bool
	mu     sync.Mutex
	key    *ecdsa.PrivateKey
	// value: how many signatures it made.
	Signed int
}

func (k *DeviceKey) private() (*ecdsa.PrivateKey, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.Absent {
		return nil, &contract.NoDeviceKeyError{Why: "no secure key store"}
	}
	if k.key == nil {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		k.key = key
	}
	return k.key, nil
}

func (k *DeviceKey) Public(context.Context) (*ecdsa.PublicKey, error) {
	key, err := k.private()
	if err != nil {
		return nil, err
	}
	return &key.PublicKey, nil
}

func (k *DeviceKey) Sign(_ context.Context, digest []byte) ([]byte, error) {
	key, err := k.private()
	if err != nil {
		return nil, err
	}
	k.mu.Lock()
	k.Signed++
	k.mu.Unlock()
	return ecdsa.SignASN1(rand.Reader, key, digest)
}
