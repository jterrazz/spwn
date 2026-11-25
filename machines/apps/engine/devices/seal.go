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
	// invariant: every entry of a book, either trusted (its seal verifies
	// against this Mac's device key) or broken, never both; a broken entry
	// reaches nothing until it is added again.
	Trusted []Device
	Broken  []Device
}

func Check(b Book, mac string, own *ecdsa.PublicKey) Checked {
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
