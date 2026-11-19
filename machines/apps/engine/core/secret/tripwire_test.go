package secret_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

func armed(t *testing.T, secrets ...string) *[]string {
	t.Helper()
	var tripped []string
	secret.Arm(secrets, func(said string) { tripped = append(tripped, said) })
	t.Cleanup(secret.Disarm)
	return &tripped
}

func TestAnArmedTripwireTripsOnAWorldSecretOrAnIssuedToken(t *testing.T) {
	tripped := armed(t, "fake-claude-token-of-this-mac")
	secret.Tripwire([]byte(`{"detail":"token fake-claude-token-of-this-mac"}`))
	secret.Tripwire([]byte("key " + token("g", "hp_A1b2C3")))
	secret.TripwireJSON(map[string]string{"k": "fake-claude-token-of-this-mac"})
	secret.Tripwire([]byte("commit " + strings.Repeat("0a1b2c3d4e", 4) + ", /Users/x"))
	if len(*tripped) != 3 {
		t.Fatalf("tripped %d times, want 3: %q", len(*tripped), *tripped)
	}
	for _, said := range *tripped {
		if strings.Contains(said, "fake-claude-token-of-this-mac") || strings.Contains(said, "hp_A1b2C3") {
			t.Errorf("the trip said the secret: %q", said)
		}
	}
}

func TestADisarmedTripwireLetsEverythingThrough(t *testing.T) {
	tripped := armed(t, "fake-claude-token-of-this-mac")
	secret.Disarm()
	secret.Tripwire([]byte("fake-claude-token-of-this-mac"))
	if len(*tripped) != 0 {
		t.Errorf("a disarmed tripwire tripped: %q", *tripped)
	}
}

func TestASecretTripsInEachFormAJSONEncoderWritesIt(t *testing.T) {
	const s = `pa"ss<word>&\\`
	tripped := armed(t, s)
	secret.TripwireJSON(map[string]string{"k": s})
	var b strings.Builder
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	_ = e.Encode(map[string]string{"k": s})
	secret.Tripwire([]byte(b.String()))
	if len(*tripped) != 2 {
		t.Fatalf("tripped %d times, want 2: %q", len(*tripped), *tripped)
	}
	for _, said := range *tripped {
		if strings.Contains(said, "word") {
			t.Errorf("the trip said the secret: %q", said)
		}
	}
}
