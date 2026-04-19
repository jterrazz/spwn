package manifest

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// FuzzManifestUnmarshal hammers the project manifest parser with
// arbitrary bytes. It must never panic — malformed input should
// surface as a yaml error, not a runtime crash.
func FuzzManifestUnmarshal(f *testing.F) {
	f.Add([]byte(`version: 1
name: my-proj
worlds:
  default:
    agents: [neo]
    workspaces: [.]
dependencies: ["spwn:unix"]
`))
	f.Add([]byte(`version: 1
name: empty`))
	f.Add([]byte(`worlds: "not a map"`))
	f.Add([]byte(`---
{invalid-yaml`))
	f.Add([]byte(``))
	f.Add([]byte(`name: "binary\x00null"`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var m Manifest
		_ = yaml.Unmarshal(data, &m)
	})
}
