package macos

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"io/fs"
	"path"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const signReason = "sign in over ssh as this Mac"

func DeviceKeyBlob(state string) string { return path.Join(state, "device-key.enclave") }

type DeviceKey struct {
	// value: this Mac's device key is kept as the blob its Secure Enclave
	// wrapped it in: only that Enclave can use it, so the file opens nothing
	// elsewhere.
	files local.FS
	state string
	// value: how long the owner has to give presence on the first signature.
	within time.Duration
	mu     sync.Mutex
	// value: the blob as this process first read or made it, so the key it
	// serves stays the key it listed.
	held []byte
}

func NewDeviceKey(files local.FS, state string, within time.Duration) *DeviceKey {
	return &DeviceKey{files: files, state: state, within: within}
}

func refusal(err error) error {
	var none *darwin.NoEnclaveError
	if errors.As(err, &none) {
		return &contract.NoDeviceKeyError{Why: none.Why}
	}
	return err
}

// invariant: the blob is made once, on the first ask; a blob the Enclave
// refuses is a refusal, never replaced by a new key.
func (k *DeviceKey) blob(create bool) ([]byte, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.held != nil {
		return k.held, nil
	}
	file := DeviceKeyBlob(k.state)
	blob, err := k.files.ReadFile(file)
	if err == nil {
		k.held = blob
	}
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		return blob, err
	}
	if !create {
		return nil, errors.New("this Mac's device key is gone: " + file)
	}
	made, err := darwin.EnclaveCreate()
	if err != nil {
		return nil, refusal(err)
	}
	if mkErr := k.files.MkdirAll(k.state, 0o700); mkErr != nil {
		return nil, mkErr
	}
	if writeErr := k.files.WriteFile(file, made, 0o600); writeErr != nil {
		return nil, writeErr
	}
	k.held = made
	return made, nil
}

func (k *DeviceKey) Public(context.Context) (*ecdsa.PublicKey, error) {
	blob, err := k.blob(true)
	if err != nil {
		return nil, err
	}
	point, err := darwin.EnclavePublic(blob)
	if err != nil {
		return nil, refusal(err)
	}
	return ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
}

func (k *DeviceKey) Sign(_ context.Context, message []byte) ([]byte, error) {
	blob, err := k.blob(false)
	if err != nil {
		return nil, err
	}
	sig, err := darwin.EnclaveSign(blob, signReason, k.within, message)
	return sig, refusal(err)
}
