package door

import (
	"context"
	"slices"
	"testing"
)

func TestTheDoorSpeaksOnlyItsPinnedAlgorithms(t *testing.T) {
	d := &Door{admit: func(context.Context, string) (string, error) { return "", errNoKey }}
	terms := d.terms(context.Background())
	for _, c := range []struct {
		what      string
		got, want []string
	}{
		{"key exchanges", terms.KeyExchanges, []string{"mlkem768x25519-sha256", "curve25519-sha256"}},
		{"ciphers", terms.Ciphers, []string{"chacha20-poly1305@openssh.com", "aes256-gcm@openssh.com", "aes128-gcm@openssh.com"}},
		{"MACs", terms.MACs, []string{"hmac-sha2-256-etm@openssh.com", "hmac-sha2-512-etm@openssh.com"}},
		{"device keys", terms.PublicKeyAuthAlgorithms, []string{"ecdsa-sha2-nistp256", "ssh-ed25519"}},
		{"signing keys", signingAlgorithms(), []string{"ecdsa-sha2-nistp256"}},
	} {
		if !slices.Equal(c.got, c.want) {
			t.Errorf("the door's %s are %v, want %v", c.what, c.got, c.want)
		}
	}
	if terms.MaxAuthTries != 2 || terms.ServerVersion != "SSH-2.0-spwn" || terms.NoClientAuth || terms.PasswordCallback != nil || terms.KeyboardInteractiveCallback != nil || terms.GSSAPIWithMICConfig != nil {
		t.Errorf("the door takes more than a device key: %+v", terms)
	}
}
