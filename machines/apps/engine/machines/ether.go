package machines

import (
	"encoding/hex"
	"strings"
)

func Ether(s string) ([6]byte, bool) {
	// value: a hardware address as macOS spells it, six hex pairs joined by
	// colons (aa:bb:cc:dd:ee:ff), or by dashes.
	var out [6]byte
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == '-' })
	if len(parts) != len(out) {
		return out, false
	}
	for i, p := range parts {
		b, err := hex.DecodeString(p)
		if err != nil || len(b) != 1 {
			return out, false
		}
		out[i] = b[0]
	}
	return out, true
}

func Spelled(a [6]byte) string {
	// value: a hardware address as macOS spells it.
	parts := make([]string, len(a))
	for i, b := range a {
		parts[i] = hex.EncodeToString([]byte{b})
	}
	return strings.Join(parts, ":")
}

func MagicPacket(ether [6]byte) []byte {
	// value: Wake-on-LAN's magic packet: six 0xFF, then the address sixteen
	// times.
	out := make([]byte, 0, 6+16*len(ether))
	for range 6 {
		out = append(out, 0xFF)
	}
	for range 16 {
		out = append(out, ether[:]...)
	}
	return out
}
