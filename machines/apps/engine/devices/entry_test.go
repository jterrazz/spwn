package devices_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/devices"
)

const ed25519Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGmY0fN5lE8XqXhcG0OB1oCw0SYqC0f6M1bM1jR5e9wz"

func deviceKey(t *testing.T) string {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	line, err := devices.LineOf(&k.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return line
}

func codeOf(err error) fail.Code {
	var e *fail.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

var added = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

func TestAnEntryKeepsItsKeyWithoutItsCommentAndItsChannelsSorted(t *testing.T) {
	key := deviceKey(t)
	d, err := devices.Entry("macbook-pro", key+" station device key", []string{"pipe", "spwn-api"}, added)
	if err != nil {
		t.Fatal(err)
	}
	if d.Key != key || !slices.Equal(d.Channels, []devices.Channel{devices.ChannelPipe, devices.ChannelAPI}) || !d.Added.Equal(added) {
		t.Fatalf("entry %+v", d)
	}
}

func TestAnEntryWithNoChannelsMayOpenAllThree(t *testing.T) {
	d, err := devices.Entry("mini", ed25519Key, nil, added)
	if err != nil || len(d.Channels) != 3 {
		t.Fatalf("entry %+v, %v; want the three channels", d, err)
	}
}

func TestAnEntryIsRefusedByTheRuleItBreaks(t *testing.T) {
	key := deviceKey(t)
	_, b64, _ := strings.Cut(key, " ")
	cases := []struct {
		name, device, key string
		channels          []string
		want              fail.Code
	}{
		{"a name that is no machine name", "Laptop Pro", key, nil, devices.BadName},
		{"an rsa key", "mini", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7", nil, devices.BadKey},
		{"a key of no type", "mini", "AAAAC3NzaC1lZDI1NTE5", nil, devices.BadKey},
		{"a key that is not base64", "mini", "ssh-ed25519 !!!", nil, devices.BadKey},
		{"a blob of another type than its line", "mini", "ssh-ed25519 " + b64, nil, devices.BadKey},
		{"a blob cut short", "mini", devices.KeyDevice + " " + b64[:40], nil, devices.BadKey},
		{"an unknown channel", "mini", key, []string{"shell"}, devices.BadChannels},
		{"a channel twice", "mini", key, []string{"pipe", "pipe"}, devices.BadChannels},
		{"hello, which is no channel", "mini", key, []string{"hello"}, devices.BadChannels},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := devices.Entry(c.device, c.key, c.channels, added)
			if got := codeOf(err); got != c.want {
				t.Fatalf("Entry = %v; want %s", err, c.want)
			}
		})
	}
}

func TestTheBookAdmitsEachNameAndKeyOnceAndNeverThisMacsKey(t *testing.T) {
	own, other := deviceKey(t), deviceKey(t)
	kept, _ := devices.Entry("macbook-pro", other, nil, added)
	book := devices.Book{Devices: []devices.Device{kept}}
	cases := []struct {
		name, device, key string
		want              fail.Code
	}{
		{"this Mac's own key", "self", own + " station device key", devices.OwnKey},
		{"a name kept already", "macbook-pro", ed25519Key, devices.Duplicate},
		{"a key kept already under another name", "laptop", other + " another comment", devices.Duplicate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := devices.Entry(c.device, c.key, nil, added)
			if err != nil {
				t.Fatal(err)
			}
			if got := codeOf(book.Admit(d, own)); got != c.want {
				t.Fatalf("Admit = %s; want %s", got, c.want)
			}
		})
	}
	fresh, _ := devices.Entry("mini", ed25519Key, nil, added)
	if err := book.Admit(fresh, own); err != nil {
		t.Fatalf("Admit = %v; want a new device admitted", err)
	}
}

func TestAFingerprintIsOpenSSHs(t *testing.T) {
	blob, _ := base64.StdEncoding.DecodeString(strings.Fields(ed25519Key)[1])
	if len(blob) == 0 || !strings.HasPrefix(devices.Fingerprint(ed25519Key), "SHA256:") || len(devices.Fingerprint(ed25519Key)) != len("SHA256:")+43 {
		t.Fatalf("fingerprint %q", devices.Fingerprint(ed25519Key))
	}
	if devices.Fingerprint("nonsense") != "" {
		t.Fatal("a line with no blob has a fingerprint")
	}
}

func TestABookFindsAndForgetsByName(t *testing.T) {
	d, _ := devices.Entry("mini", ed25519Key, nil, added)
	book := devices.Book{Devices: []devices.Device{d}}
	if got, ok := book.Find("mini"); !ok || got.Key != d.Key {
		t.Fatalf("Find = %+v, %v", got, ok)
	}
	if _, ok := book.Find("laptop"); ok || book.Forget("laptop") {
		t.Fatal("an unknown name was found or forgotten")
	}
	if !book.Forget("mini") || len(book.Devices) != 0 {
		t.Fatalf("Forget left %+v", book.Devices)
	}
}
