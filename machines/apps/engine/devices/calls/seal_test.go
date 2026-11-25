package calls_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/calls"
)

// value: a device key held in memory, as the Enclave's would be.
type memoryKey struct {
	key *ecdsa.PrivateKey
}

func newKey(t *testing.T) *memoryKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &memoryKey{key: k}
}

func (k *memoryKey) Kept(ctx context.Context) (*ecdsa.PublicKey, error) { return k.Public(ctx) }

func (k *memoryKey) Public(context.Context) (*ecdsa.PublicKey, error) {
	return &k.key.PublicKey, nil
}

func (k *memoryKey) Sign(_ context.Context, message []byte) ([]byte, error) {
	digest := sha256.Sum256(message)
	return ecdsa.SignASN1(rand.Reader, k.key, digest[:])
}

const peerKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGmY0fN5lE8XqXhcG0OB1oCw0SYqC0f6M1bM1jR5e9wz"

func sealed(t *testing.T, s calls.Sealer) devices.Device {
	t.Helper()
	d, err := devices.Entry("macbook-pro", peerKey, []string{"pipe", "spwn-api"}, time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	d, err = s.Seal(t.Context(), d)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func check(t *testing.T, s calls.Sealer, d devices.Device) devices.Checked {
	t.Helper()
	own, err := s.Key.Public(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	b := devices.Book{Generation: 1, Devices: []devices.Device{d}}
	return devices.Check(b, b.Head(), s.Mac, own)
}

func TestASealedEntryIsTrusted(t *testing.T) {
	s := calls.Sealer{Key: newKey(t), Mac: "mac-mini"}
	if got := check(t, s, sealed(t, s)); len(got.Trusted) != 1 || len(got.Broken) != 0 {
		t.Fatalf("checked %+v; want the entry trusted", got)
	}
}

func TestAnEntryWhoseSealFailsIsIgnoredAndReported(t *testing.T) {
	key := newKey(t)
	s := calls.Sealer{Key: key, Mac: "mac-mini"}
	cases := []struct {
		name  string
		check calls.Sealer
		edit  func(*devices.Device)
	}{
		{"an edited key", s, func(d *devices.Device) {
			d.Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOtherKeyOfAnotherDeviceEntirely00000000000"
		}},
		{"an edited channel list", s, func(d *devices.Device) { d.Channels = devices.AllChannels() }},
		{"an edited name", s, func(d *devices.Device) { d.Name = "laptop" }},
		{"an edited date", s, func(d *devices.Device) { d.Added = d.Added.Add(time.Second) }},
		{"no seal at all", s, func(d *devices.Device) { d.Seal = nil }},
		{"an entry of another Mac's book", calls.Sealer{Key: key, Mac: "macbook-pro"}, func(*devices.Device) {}},
		{"an entry sealed by a key swapped since", calls.Sealer{Key: newKey(t), Mac: "mac-mini"}, func(*devices.Device) {}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := sealed(t, s)
			c.edit(&d)
			got := check(t, c.check, d)
			if len(got.Trusted) != 0 || len(got.Broken) != 1 || got.Broken[0].Name != d.Name {
				t.Fatalf("checked %+v; want the entry broken", got)
			}
		})
	}
}

func TestABrokenEntryGivesWayWhenItsDeviceIsAddedAgain(t *testing.T) {
	s := calls.Sealer{Key: newKey(t), Mac: "mac-mini"}
	own, err := s.Key.Public(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	line, err := devices.LineOf(own)
	if err != nil {
		t.Fatal(err)
	}
	broken := sealed(t, s)
	broken.Channels = devices.AllChannels()
	book := devices.Book{Devices: []devices.Device{broken}}
	again := sealed(t, s)
	if err = book.Readmit(again, line, s.Mac, own); err != nil {
		t.Fatalf("readmit over a broken entry: %v", err)
	}
	if got := devices.Check(book, book.Head(), s.Mac, own); len(got.Trusted) != 1 || len(got.Broken) != 0 {
		t.Fatalf("checked %+v; want the new entry alone, trusted", got)
	}
	if err = book.Readmit(sealed(t, s), line, s.Mac, own); err == nil {
		t.Fatal("readmit over a trusted entry of the same name: want devices.duplicate")
	}
}

func TestOnlyTrustedEntriesThatMayOpenAChannelOpenIt(t *testing.T) {
	s := calls.Sealer{Key: newKey(t), Mac: "mac-mini"}
	own, err := s.Key.Public(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	piped := sealed(t, s)
	api, err := devices.Entry("tablet", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGmY0fN5lE8XqXhcG0OB1oCw0SYqC0f6M1bM1jR5e9wz", []string{"spwn-api"}, piped.Added)
	if err != nil {
		t.Fatal(err)
	}
	if api, err = s.Seal(t.Context(), api); err != nil {
		t.Fatal(err)
	}
	broken := piped
	broken.Name = "stranger"
	book := devices.Book{Generation: 1, Devices: []devices.Device{piped, api, broken}}
	got := devices.Check(book, book.Head(), s.Mac, own).Opening(devices.ChannelPipe)
	if want := []string{piped.Key}; !slices.Equal(got, want) {
		t.Fatalf("pipe keys %q, want %q", got, want)
	}
}
