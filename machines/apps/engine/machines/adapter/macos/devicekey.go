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

func DeviceKeyBlob(state string, access darwin.Access) string {
	// value: the file a device key of this access is kept in, so the file
	// names who the key signs for.
	if access == darwin.AccessUnattended {
		return path.Join(state, "device-key-unattended.enclave")
	}
	return path.Join(state, "device-key.enclave")
}

type enclave interface {
	create(access darwin.Access) ([]byte, error)
	public(blob []byte) ([]byte, error)
	sign(blob []byte, access darwin.Access, reason string, within time.Duration, message []byte) ([]byte, error)
}

type secureEnclave struct{}

func (secureEnclave) create(access darwin.Access) ([]byte, error) {
	return darwin.EnclaveCreate(access)
}
func (secureEnclave) public(blob []byte) ([]byte, error) { return darwin.EnclavePublic(blob) }
func (secureEnclave) sign(blob []byte, access darwin.Access, reason string, within time.Duration, message []byte) ([]byte, error) {
	return darwin.EnclaveSign(blob, access, reason, within, message)
}

type DeviceKey struct {
	// value: this Mac's device key is kept as the blob its Secure Enclave
	// wrapped it in: only that Enclave can use it, so the file opens nothing
	// elsewhere.
	files   local.FS
	state   string
	enclave enclave
	// value: who this process's key signs for, as the Mac is run when it
	// starts.
	access darwin.Access
	// value: how long the owner has to give presence on the first signature.
	within time.Duration
	mu     sync.Mutex
	// value: the blob as this process first read or made it, so the key it
	// serves stays the key it listed, and its public half once the Enclave
	// gave it: neither changes for the life of the process.
	held []byte
	pub  *ecdsa.PublicKey
}

func NewDeviceKey(files local.FS, state string, within time.Duration, access darwin.Access) *DeviceKey {
	return &DeviceKey{files: files, state: state, within: within, access: access, enclave: secureEnclave{}}
}

func refusal(err error) error {
	var none *darwin.NoEnclaveError
	if errors.As(err, &none) {
		return &contract.NoDeviceKeyError{Why: none.Why}
	}
	return err
}

// invariant: the blob is made once per access, on the first ask; the key of
// the other access is deleted once its replacement is kept, since the Enclave
// never changes a key's access. A blob the Enclave refuses is a refusal,
// never replaced by a new key.
func (k *DeviceKey) blob(create bool) ([]byte, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.held != nil {
		return k.held, nil
	}
	file := DeviceKeyBlob(k.state, k.access)
	blob, err := k.files.ReadFile(file)
	if err == nil {
		k.held = blob
	}
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		return blob, err
	}
	if !create {
		return nil, &goneError{file: file}
	}
	made, err := k.enclave.create(k.access)
	if err != nil {
		return nil, refusal(err)
	}
	if mkErr := k.files.MkdirAll(k.state, 0o700); mkErr != nil {
		return nil, mkErr
	}
	if writeErr := k.files.WriteFile(file, made, 0o600); writeErr != nil {
		return nil, writeErr
	}
	other := DeviceKeyBlob(k.state, darwin.AccessPresent)
	if k.access == darwin.AccessPresent {
		other = DeviceKeyBlob(k.state, darwin.AccessUnattended)
	}
	if rmErr := k.files.Remove(other); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
		return nil, rmErr
	}
	k.held = made
	return made, nil
}

func (k *DeviceKey) Public(context.Context) (*ecdsa.PublicKey, error) {
	return k.public(true)
}

func (k *DeviceKey) Kept(context.Context) (*ecdsa.PublicKey, error) {
	return k.public(false)
}

func (k *DeviceKey) public(create bool) (*ecdsa.PublicKey, error) {
	blob, err := k.blob(create)
	var gone *goneError
	if errors.As(err, &gone) {
		return nil, &contract.NoDeviceKeyError{Why: "none is kept yet in " + gone.file + "; the engine makes it at its start"}
	}
	if err != nil {
		return nil, err
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.pub != nil {
		return k.pub, nil
	}
	point, err := k.enclave.public(blob)
	if err != nil {
		return nil, refusal(err)
	}
	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
	if err != nil {
		return nil, err
	}
	k.pub = pub
	return pub, nil
}

type goneError struct{ file string }

func (e *goneError) Error() string { return "this Mac's device key is gone: " + e.file }

func (k *DeviceKey) Sign(_ context.Context, message []byte) ([]byte, error) {
	blob, err := k.blob(false)
	if err != nil {
		return nil, err
	}
	sig, err := k.enclave.sign(blob, k.access, signReason, k.within, message)
	return sig, refusal(err)
}
