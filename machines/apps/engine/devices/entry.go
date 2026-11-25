package devices

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

const (
	KeyDevice  = "ecdsa-sha2-nistp256"
	KeyEd25519 = "ssh-ed25519"
	curve      = "nistp256"
)

func readString(blob []byte) (s, rest []byte, ok bool) {
	if len(blob) < 4 {
		return nil, nil, false
	}
	n := int64(binary.BigEndian.Uint32(blob))
	if n > int64(len(blob)-4) {
		return nil, nil, false
	}
	return blob[4 : 4+n], blob[4+n:], true
}

func appendString(out, s []byte) []byte {
	out = binary.BigEndian.AppendUint32(out, uint32(len(s))) //nolint:gosec // outside: a key blob's strings are a type name and a point, never 4 GiB.
	return append(out, s...)
}

func validBlob(kind string, blob []byte) bool {
	// invariant: a key blob is read whole, as RFC 4253 §6.6 and RFC 5656 §3.1
	// lay it out: its type, then its point; a byte left over refuses it.
	got, body, ok := readString(blob)
	if !ok || string(got) != kind {
		return false
	}
	switch kind {
	case KeyEd25519:
		point, rest, ok := readString(body)
		return ok && len(point) == 32 && len(rest) == 0
	case KeyDevice:
		name, after, ok := readString(body)
		if !ok || string(name) != curve {
			return false
		}
		point, rest, ok := readString(after)
		if !ok || len(rest) != 0 {
			return false
		}
		_, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
		return err == nil
	}
	return false
}

func ParseKey(line string) (string, error) {
	// invariant: a public key line is "<type> <base64> [comment]"; the key
	// kept is its type and base64 alone, so a comment never makes two keys.
	fields := strings.Fields(line)
	if len(fields) < 2 || (fields[0] != KeyDevice && fields[0] != KeyEd25519) {
		return "", Refusal(fail.Refused, BadKey, line)
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil || !validBlob(fields[0], blob) {
		return "", Refusal(fail.Refused, BadKey, line)
	}
	return fields[0] + " " + fields[1], nil
}

func LineOf(pub *ecdsa.PublicKey) (string, error) {
	point, err := pub.Bytes()
	if err != nil {
		return "", err
	}
	blob := appendString(appendString(appendString(nil, []byte(KeyDevice)), []byte(curve)), point)
	return KeyDevice + " " + base64.StdEncoding.EncodeToString(blob), nil
}

func Fingerprint(key string) string {
	// value: OpenSSH's fingerprint of a key: SHA256: and the unpadded base64
	// of its blob's digest; empty for a line that holds no blob.
	_, b64, ok := strings.Cut(key, " ")
	if !ok {
		return ""
	}
	b64, _, _ = strings.Cut(b64, " ")
	blob, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

func ParseChannels(words []string) ([]Channel, error) {
	// invariant: no word asks for every channel; each word names one, once,
	// and the channels are kept sorted.
	if len(words) == 0 {
		return AllChannels(), nil
	}
	var out []Channel
	for _, w := range words {
		c := Channel(strings.TrimSpace(w))
		if !slices.Contains(AllChannels(), c) || slices.Contains(out, c) {
			return nil, Refusal(fail.Refused, BadChannels, strings.Join(words, ","))
		}
		out = append(out, c)
	}
	slices.Sort(out)
	return out, nil
}

func Entry(name, key string, channels []string, added time.Time) (Device, error) {
	if !setup.ValidName(name) {
		return Device{}, Refusal(fail.Refused, BadName, name)
	}
	parsed, err := ParseKey(key)
	if err != nil {
		return Device{}, err
	}
	chosen, err := ParseChannels(channels)
	if err != nil {
		return Device{}, err
	}
	slices.Sort(chosen)
	return Device{Name: name, Key: parsed, Channels: chosen, Added: added.UTC().Truncate(time.Second)}, nil
}

func (b Book) Admit(d Device, own string) error {
	// invariant: a book holds each name once and each key once, and never
	// this machine's own device key, which its VMs trust already.
	if own != "" && sameKey(d.Key, own) {
		return Refusal(fail.Refused, OwnKey, d.Name)
	}
	for _, kept := range b.Devices {
		if kept.Name == d.Name || sameKey(kept.Key, d.Key) {
			e := Refusal(fail.Refused, Duplicate, d.Name)
			e.Detail = map[string]string{"device": kept.Name}
			return e
		}
	}
	return nil
}

func sameKey(a, b string) bool {
	fa, fb := strings.Fields(a), strings.Fields(b)
	return len(fa) >= 2 && len(fb) >= 2 && fa[0] == fb[0] && fa[1] == fb[1]
}
