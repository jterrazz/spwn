package macos

import (
	"bytes"
	"testing"
)

func TestEncodeKCPasswordPadsToTheCipherCycle(t *testing.T) {
	got := encodeKCPassword("")
	want := []byte{0x7D, 0x89, 0x52, 0x23, 0xD2, 0xBC, 0xDD, 0xEA, 0xA3, 0xB9, 0x1F, 0x7D}
	if !bytes.Equal(got, want) {
		t.Fatalf("empty password: got %x, want %x", got, want)
	}
}

func TestEncodeKCPasswordRoundTripsAtEveryLengthAroundTheBlock(t *testing.T) {
	for _, pw := range []string{"a", "hello", "12345678901", "12-character", "exactly-12-c", "this-is-a-longer-test-string-that-spans"} {
		out := encodeKCPassword(pw)
		if len(out)%12 != 0 || len(out) <= len(pw) {
			t.Errorf("%q: length %d is not the next multiple of 12 above the password", pw, len(out))
		}
		decoded := make([]byte, len(out))
		for i, b := range out {
			decoded[i] = b ^ kcpasswordCipher[i%len(kcpasswordCipher)]
		}
		if !bytes.HasPrefix(decoded, []byte(pw)) {
			t.Errorf("%q: decodes to %q", pw, decoded)
		}
	}
}
