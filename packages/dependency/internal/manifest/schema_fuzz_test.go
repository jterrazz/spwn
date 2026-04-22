package manifest

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
  packages:
    apt: [bash, coreutils]`))
	f.Add([]byte(`name: matrix
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
