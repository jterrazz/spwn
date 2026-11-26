package fake

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
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

func DeviceKeyOf(key *ecdsa.PrivateKey) *DeviceKey { return &DeviceKey{key: key} }

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

func (k *DeviceKey) Kept(context.Context) (*ecdsa.PublicKey, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.Absent || k.key == nil {
		return nil, &contract.NoDeviceKeyError{Why: "none is kept yet"}
	}
	return &k.key.PublicKey, nil
}

func (k *DeviceKey) Sign(_ context.Context, message []byte) ([]byte, error) {
	key, err := k.private()
	if err != nil {
		return nil, err
	}
	k.mu.Lock()
	k.Signed++
	k.mu.Unlock()
	digest := sha256.Sum256(message)
	return ecdsa.SignASN1(rand.Reader, key, digest[:])
}

func DeviceKeyFor(mac string) *DeviceKey {
	// invariant: a world Mac's device key is the same in every process of a
	// story, made from its name alone, so what one run seals the next one
	// checks.
	return keyFrom("world device key\n" + mac)
}

func DoorKeyFor(mac string) *DeviceKey {
	// value: a world Mac's door host key, its own and never its device key,
	// the same in every process of a story.
	return keyFrom("world door key\n" + mac)
}

func keyFrom(seed string) *DeviceKey {
	scalar := sha256.Sum256([]byte(seed))
	key, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), scalar[:])
	if err != nil {
		return &DeviceKey{}
	}
	return DeviceKeyOf(key)
}
