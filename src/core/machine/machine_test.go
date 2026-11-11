package machine_test

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/src/core/machine"
)

func TestAnOlderStationsMacReadsAsAComputer(t *testing.T) {
	for _, word := range []string{"computer", "mac"} {
		var k machine.Kind
		if err := json.Unmarshal([]byte(`"`+word+`"`), &k); err != nil || k != machine.Computer {
			t.Fatalf("%q read as %q, %v", word, k, err)
		}
	}
	for _, kind := range []machine.Kind{machine.Server, machine.Workspace} {
		var k machine.Kind
		if err := json.Unmarshal([]byte(`"`+string(kind)+`"`), &k); err != nil || k != kind {
			t.Fatalf("%q read as %q, %v", kind, k, err)
		}
	}
	out, _ := json.Marshal(machine.Computer)
	if string(out) != `"computer"` {
		t.Fatalf("a Computer is written %s, never the older word", out)
	}
}
