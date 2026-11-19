package contracttest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func DeviceKey(t *testing.T, subject func(t *testing.T) contract.DeviceKey) {
	t.Run("the key is a P-256 key, the same at every ask", func(t *testing.T) {
		k := subject(t)
		first, err := k.Public(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if first.Curve != elliptic.P256() {
			t.Fatalf("curve = %s; want P-256", first.Curve.Params().Name)
		}
		again, err := k.Public(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !first.Equal(again) {
			t.Fatal("a second ask gave another key")
		}
	})
	t.Run("a signature verifies against the public half, and only for its digest", func(t *testing.T) {
		k := subject(t)
		digest := sha256.Sum256([]byte("a challenge"))
		sig, err := k.Sign(t.Context(), digest[:])
		if err != nil {
			t.Fatal(err)
		}
		pub, err := k.Public(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !ecdsa.VerifyASN1(pub, digest[:], sig) {
			t.Fatal("the signature does not verify against the key's public half")
		}
		other := sha256.Sum256([]byte("another challenge"))
		if ecdsa.VerifyASN1(pub, other[:], sig) {
			t.Fatal("the signature verifies another digest")
		}
	})
}
