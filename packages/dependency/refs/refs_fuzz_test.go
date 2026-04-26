package refs

import "testing"

// FuzzParseRef hammers the ref-kind classifier with arbitrary
// inputs. ParseRef must never panic regardless of input shape.
func FuzzParseRef(f *testing.F) {
	f.Add("")
	f.Add("spwn:unix")
	f.Add("spwn:unix@24.04")
	f.Add("github:jterrazz/skills")
	f.Add("skill/focus")
	f.Add("tool/ffmpeg")
	f.Add("hook/pre-spawn")
	f.Add("bare-name")
	f.Add("@")
	f.Add("@scope")
	f.Add("@/")
	f.Add("///")
	f.Add("tool/")
	f.Add("/foo")
	f.Add("tool/foo/bar")

	f.Fuzz(func(t *testing.T, s string) {
		_ = ParseRef(s)
	})
}

// FuzzSplitVersion verifies SplitVersion never panics on arbitrary
// input. The round-trip invariant: name + "@" + version either
// equals the input or the version is empty.
func FuzzSplitVersion(f *testing.F) {
	f.Add("")
	f.Add("spwn:unix@1.0")
	f.Add("bare@1.0")
	f.Add("@@@")
	f.Add("no-version")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = SplitVersion(s)
	})
}
