package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/src/ports/contract"
)

func Tailnet(t *testing.T, open func(t *testing.T) contract.Tailnet) {
	// value: open answers a node signed in to its tailnet, sharing on, sharing
	// nothing yet.
	ctx := context.Background()

	t.Run("a node with sharing on names the DNS name its shares answer at", func(t *testing.T) {
		tn := open(t)
		ready, err := tn.Ready(ctx)
		if err != nil || !ready.On || ready.DNSName == "" {
			t.Fatalf("Ready = %+v, %v; want sharing on at a DNS name", ready, err)
		}
		self, err := tn.Self(ctx)
		if err != nil || self.Sharing != ready || self.DNSName != ready.DNSName {
			t.Fatalf("Self = %+v, %v; want the sharing Ready reported, %+v", self, err, ready)
		}
	})

	t.Run("warming the certificate of a node sharing on succeeds", func(t *testing.T) {
		tn := open(t)
		if _, err := tn.Ready(ctx); err != nil {
			t.Fatalf("Ready: %v", err)
		}
		if err := tn.Warm(ctx); err != nil {
			t.Fatalf("Warm: %v", err)
		}
	})
}
