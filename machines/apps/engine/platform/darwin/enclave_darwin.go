//go:build darwin && cgo

package darwin

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../../.artifacts/enclave -lstationenclave -L/usr/lib/swift -framework CryptoKit -framework LocalAuthentication -framework Security -framework Foundation
#include <stdlib.h>
#include <stdint.h>
int station_enclave_create(uint8_t **out, long *n);
int station_enclave_public(const uint8_t *blob, long bn, uint8_t **out, long *n);
int station_enclave_sign(const uint8_t *blob, long bn, const char *reason, double within, const uint8_t *message, long mn, uint8_t **out, long *n);
*/
import "C"

import (
	"errors"
	"sync"
	"time"
	"unsafe"
)

// invariant: one signature at a time, so a session's first ones ask the
// owner's presence once.
var signing sync.Mutex

func taken(rc C.int, out *C.uint8_t, n C.long) ([]byte, error) {
	b := C.GoBytes(unsafe.Pointer(out), C.int(n))
	C.free(unsafe.Pointer(out))
	switch rc {
	case 0:
		return b, nil
	case 2:
		return nil, &NoEnclaveError{Why: string(b)}
	default:
		return nil, errors.New(string(b))
	}
}

func bytesOf(b []byte) (*C.uint8_t, C.long) {
	if len(b) == 0 {
		return nil, 0
	}
	return (*C.uint8_t)(unsafe.Pointer(&b[0])), C.long(len(b))
}

func EnclaveCreate() ([]byte, error) {
	var out *C.uint8_t
	var n C.long
	return taken(C.station_enclave_create(&out, &n), out, n)
}

func EnclavePublic(blob []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, errors.New("this Mac's device key is empty")
	}
	var out *C.uint8_t
	var n C.long
	b, bn := bytesOf(blob)
	return taken(C.station_enclave_public(b, bn, &out, &n), out, n)
}

func EnclaveSign(blob []byte, reason string, within time.Duration, message []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, errors.New("this Mac's device key is empty")
	}
	if len(message) == 0 {
		return nil, errors.New("nothing to sign")
	}
	text := C.CString(reason)
	defer C.free(unsafe.Pointer(text))
	signing.Lock()
	defer signing.Unlock()
	var out *C.uint8_t
	var n C.long
	b, bn := bytesOf(blob)
	m, mn := bytesOf(message)
	return taken(C.station_enclave_sign(b, bn, text, C.double(within.Seconds()), m, mn, &out, &n), out, n)
}
