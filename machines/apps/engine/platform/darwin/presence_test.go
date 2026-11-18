//go:build !darwin || !cgo

package darwin

import (
	"testing"
	"time"
)

func TestABuildWithoutLocalAuthenticationHasNoPresenceToAsk(t *testing.T) {
	if got := AskPresence("give a password", time.Second); got != PresenceAbsent {
		t.Fatalf("AskPresence = %d; want absent, never a yes", got)
	}
}
