package devices_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/devices"
)

func sealedBy(t *testing.T, own *ecdsa.PrivateKey, mac string, d devices.Device) devices.Device {
	t.Helper()
	digest := sha256.Sum256(devices.SealMessage(mac, &own.PublicKey, d))
	seal, err := ecdsa.SignASN1(rand.Reader, own, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	d.Seal = seal
	return d
}

func TestTheHolderOfAKeyIsTheTrustedEntryOfThatKeyAlone(t *testing.T) {
	own, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	laptop := sealedBy(t, own, "mini", devices.Device{Name: "laptop", Key: deviceKey(t), Channels: devices.AllChannels(), Added: added})
	phone := sealedBy(t, own, "mini", devices.Device{Name: "phone", Key: deviceKey(t), Channels: devices.AllChannels(), Added: added})
	broken := devices.Device{Name: "tablet", Key: deviceKey(t), Channels: devices.AllChannels(), Added: added, Seal: []byte("not a seal")}
	book := devices.Book{Generation: 3, Devices: []devices.Device{laptop, phone, broken}}
	checked := devices.Check(book, book.Head(), "mini", &own.PublicKey)
	if len(checked.Broken) != 1 || checked.Stale {
		t.Fatalf("the book checks as %+v, want tablet alone broken", checked)
	}
	for _, c := range []struct {
		key, want string
	}{
		{laptop.Key, "laptop"},
		{laptop.Key + " a comment", "laptop"},
		{phone.Key, "phone"},
	} {
		if d, ok := checked.Holder(c.key); !ok || d.Name != c.want {
			t.Errorf("the holder of %s's key is %q %v", c.want, d.Name, ok)
		}
	}
	for what, key := range map[string]string{
		"a key no entry names":         deviceKey(t),
		"the key of a broken entry":    broken.Key,
		"a key type alone":             "ecdsa-sha2-nistp256",
		"no key":                       "",
		"the right key under its type": "ssh-ed25519 " + laptop.Key[len("ecdsa-sha2-nistp256 "):],
	} {
		if d, ok := checked.Holder(key); ok {
			t.Errorf("%s is held by %q", what, d.Name)
		}
	}
	stale := devices.Check(book, devices.Head{}, "mini", &own.PublicKey)
	if d, ok := stale.Holder(laptop.Key); ok {
		t.Errorf("a book that is not the anchored one lets %q hold its key", d.Name)
	}
}
