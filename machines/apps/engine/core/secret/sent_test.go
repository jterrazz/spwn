package secret_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

func TestASentSecretReadsFromJSONAndNeverPrintsItsBytes(t *testing.T) {
	var in struct{ Code secret.Sent }
	if err := json.Unmarshal([]byte(`{"Code":"hunter-two"}`), &in); err != nil {
		t.Fatal(err)
	}
	if got := string(in.Code.Reveal()); got != "hunter-two" {
		t.Fatalf("Reveal = %q, want the typed text", got)
	}
	back, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, shown := range []string{string(back), fmt.Sprint(in), fmt.Sprintf("%+v", in), fmt.Sprintf("%#v", in.Code), in.Code.String()} {
		if shown == "" || strings.Contains(shown, "hunter-two") {
			t.Errorf("a sent secret printed %q", shown)
		}
	}
}

func TestASentSecretRefusesANonString(t *testing.T) {
	var in struct{ Code secret.Sent }
	if err := json.Unmarshal([]byte(`{"Code":12}`), &in); err == nil {
		t.Fatal("a number was read as a secret")
	}
}
