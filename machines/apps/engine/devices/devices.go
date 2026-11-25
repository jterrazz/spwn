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
	// added; this machine's own device key is never in it.
	Devices []Device `json:"devices"`
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
