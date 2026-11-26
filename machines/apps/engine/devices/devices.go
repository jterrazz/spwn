package devices

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Channel string

const (
	ChannelAPI       Channel = "spwn-api"
	ChannelPipe      Channel = "pipe"
	ChannelTerminals Channel = catalog.TerminalsChannel
)

func AllChannels() []Channel { return []Channel{ChannelAPI, ChannelPipe, ChannelTerminals} }

type Device struct {
	// invariant: another device this machine lets reach it: its name, its
	// public key as one "<type> <base64>" line with no comment, the
	// channels it may open, sorted, and when it was added, in UTC.
	Name     string    `json:"name"`
	Key      string    `json:"key"`
	Channels []Channel `json:"channels"`
	Added    time.Time `json:"added"`
	// value: this Mac's device key's ASN.1 ECDSA signature over the entry's
	// SealMessage; an entry whose seal does not verify is never trusted.
	Seal []byte `json:"seal,omitempty"`
}

type Book struct {
	// invariant: one machine's device book, in the order the entries were
	// added; this machine's own device key is never in it. Generation is
	// the change that wrote it, 0 for a book never written.
	Generation int      `json:"generation,omitempty"`
	Devices    []Device `json:"devices"`
}

func (b Book) Find(name string) (Device, bool) {
	i := slices.IndexFunc(b.Devices, func(d Device) bool { return d.Name == name })
	if i < 0 {
		return Device{}, false
	}
	return b.Devices[i], true
}

func (b *Book) Forget(name string) bool {
	before := len(b.Devices)
	b.Devices = slices.DeleteFunc(b.Devices, func(d Device) bool { return d.Name == name })
	return len(b.Devices) != before
}

type Row struct {
	// value: an entry as a reader sees it: its key's fingerprint beside the
	// key, and never its seal; a broken entry's key is not given, only its
	// fingerprint.
	Name        string    `json:"name"`
	Key         string    `json:"key,omitempty"`
	Fingerprint string    `json:"fingerprint"`
	Channels    []Channel `json:"channels"`
	Added       time.Time `json:"added"`
}

func RowOf(d Device) Row {
	return Row{Name: d.Name, Key: d.Key, Fingerprint: Fingerprint(d.Key), Channels: d.Channels, Added: d.Added}
}

func BrokenRowOf(d Device) Row {
	// invariant: a broken entry's key is whatever wrote the book wrote, so
	// it is never handed back as the key to add again: that key is taken
	// from the device itself.
	r := RowOf(d)
	r.Key = ""
	return r
}

type Listing struct {
	// invariant: Devices are the entries this Mac trusts, Broken the ones
	// whose seal failed, which reach nothing until they are added again;
	// both are empty lists, never absent.
	Mac string `json:"mac"`
	// value: the book is not the one this Mac last wrote (a copy from
	// before, written back, or a book this Mac never wrote): every entry is
	// broken, and the first add starts a new book.
	Stale bool `json:"stale"`
	// value: this Mac's own device key line, the one another device adds;
	// empty on a Mac that has nowhere to keep one.
	Key     string `json:"key"`
	Devices []Row  `json:"devices"`
	Broken  []Row  `json:"broken"`
}
