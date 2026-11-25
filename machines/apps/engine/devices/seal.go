package devices

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"slices"
	"strings"
	"time"
)

const sealDomain = "spwn-device-book/v1"

func SealMessage(mac string, own *ecdsa.PublicKey, d Device) []byte {
	// invariant: what a seal signs names this Mac by its name and by the
	// digest of its device key, then every field of the entry, so an entry
	// copied from another Mac, sealed by a key swapped since, or edited by
	// hand never verifies.
	point, err := own.Bytes()
	if err != nil {
		return nil
	}
	sum := sha256.Sum256(point)
	channels := make([]string, 0, len(d.Channels))
	for _, c := range d.Channels {
		channels = append(channels, string(c))
	}
	slices.Sort(channels)
	return []byte(strings.Join([]string{
		sealDomain, mac, base64.StdEncoding.EncodeToString(sum[:]),
		d.Name, d.Key, strings.Join(channels, ","), d.Added.UTC().Format(time.RFC3339),
	}, "\n"))
}

func Sealed(mac string, own *ecdsa.PublicKey, d Device) bool {
	msg := SealMessage(mac, own, d)
	if msg == nil || len(d.Seal) == 0 {
		return false
	}
	digest := sha256.Sum256(msg)
	return ecdsa.VerifyASN1(own, digest[:], d.Seal)
}

type Checked struct {
	// invariant: every entry of a book, either trusted (the book is the one
	// this Mac last wrote and the entry's seal verifies against this Mac's
	// device key) or broken, never both; a broken entry reaches nothing
	// until it is added again. Stale is a book that is not the anchored
	// one, every entry of it broken.
	Trusted []Device
	Broken  []Device
	Stale   bool
}

func Check(b Book, anchored Head, mac string, own *ecdsa.PublicKey) Checked {
	if !b.Anchored(anchored) {
		return Checked{Trusted: []Device{}, Broken: clone(b.Devices), Stale: true}
	}
	return sealed(b, mac, own)
}

func sealed(b Book, mac string, own *ecdsa.PublicKey) Checked {
	out := Checked{Trusted: []Device{}, Broken: []Device{}}
	for _, d := range b.Devices {
		if own != nil && Sealed(mac, own, d) {
			out.Trusted = append(out.Trusted, d)
			continue
		}
		out.Broken = append(out.Broken, d)
	}
	return out
}

func (b Book) Admits(d Device, own, mac string, key *ecdsa.PublicKey) error {
	// invariant: b is the book a change starts from (Base); a broken entry
	// reaches nothing, so it never blocks adding its device again: the
	// entries whose seal holds judge the new one, which then takes the
	// place of every broken entry of its name or its key.
	trusted := Book{Devices: sealed(b, mac, key).Trusted}
	return trusted.Admit(d, own)
}

func (b *Book) Readmit(d Device, own, mac string, key *ecdsa.PublicKey) error {
	if err := b.Admits(d, own, mac, key); err != nil {
		return err
	}
	b.Devices = append(slices.DeleteFunc(slices.Clone(b.Devices), func(kept Device) bool {
		return kept.Name == d.Name || sameKey(kept.Key, d.Key)
	}), d)
	return nil
}

func (c Checked) Opening(ch Channel) []string {
	var keys []string
	for _, d := range c.Trusted {
		if slices.Contains(d.Channels, ch) {
			keys = append(keys, d.Key)
		}
	}
	return keys
}
