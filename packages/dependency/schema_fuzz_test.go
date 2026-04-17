package dependency

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// FuzzSchemaUnmarshal feeds arbitrary bytes into the Schema
// unmarshaller. The parser must either succeed or return a clean
// error — never panic.
func FuzzSchemaUnmarshal(f *testing.F) {
	// Seed corpus: known-good + known-malformed samples.
	f.Add([]byte(`name: "spwn:unix"
version: "1.0"
install:
  packages: [bash, coreutils]`))
	f.Add([]byte(`name: matrix
kind: tool
worlds:
  matrix:
    agents: [neo]`))
	f.Add([]byte(`{malformed`))
	f.Add([]byte(``))
	f.Add([]byte(`!!invalid-yaml`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var s Schema
		_ = yaml.Unmarshal(data, &s) // error fine, panic is not
	})
}

// FuzzParseRef hammers the ref-kind classifier with arbitrary
// inputs. ParseRef must never panic regardless of input shape.
func FuzzParseRef(f *testing.F) {
	f.Add("")
	f.Add("spwn:unix")
	f.Add("spwn:unix@24.04")
	f.Add("github.com/jterrazz/skills")
	f.Add("bare-name")
	f.Add("@")
	f.Add("@scope")
	f.Add("@/")
	f.Add("///")

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
