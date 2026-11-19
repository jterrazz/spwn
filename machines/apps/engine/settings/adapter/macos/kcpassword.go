package macos

// outside: the public XOR cipher loginwindow uses to obfuscate /etc/kcpassword, stable
// since Mac OS X 10.4: obfuscation, not a secret.
var kcpasswordCipher = [...]byte{
	0x7D, 0x89, 0x52, 0x23, 0xD2, 0xBC, 0xDD, 0xEA, 0xA3, 0xB9, 0x1F,
}

// outside: each byte XORs with cipher[i % len], padded with the cycle to a multiple of
// 12 and at least 12 bytes, so loginwindow finds the end of a 12-multiple password.
func encodeKCPassword(password string) []byte {
	pw := []byte(password)
	out := make([]byte, len(pw), len(pw)+12)
	for i, b := range pw {
		out[i] = b ^ kcpasswordCipher[i%len(kcpasswordCipher)]
	}
	target := ((len(out) / 12) + 1) * 12
	for len(out) < target {
		out = append(out, kcpasswordCipher[len(out)%len(kcpasswordCipher)])
	}
	return out
}
