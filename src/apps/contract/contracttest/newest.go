package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
)

func Newest(t *testing.T, sample catalog.Install, build Build) {
	// invariant: what every Newest owes an upgrade within a range: a
	// version for an install its manager holds, and an error, never a
	// guess, where its manager is missing.
	t.Helper()
	ctx := context.Background()
	reads := func(t *testing.T, seed Seed) contract.Newest {
		t.Helper()
		n, ok := build(t, seed).(contract.Newest)
		if !ok {
			t.Fatalf("%T reads no newest version", build(t, seed))
		}
		return n
	}
	t.Run("answers a version", func(t *testing.T) {
		v, err := reads(t, Seed{Installed: map[string]string{sample.Key(): "1.0.0"}}).Newest(ctx, sample)
		if err != nil || v == "" {
			t.Errorf("Newest = %q, %v; want a version", v, err)
		}
	})
	t.Run("a missing manager is an error", func(t *testing.T) {
		if v, err := reads(t, Seed{Absent: true}).Newest(ctx, sample); err == nil {
			t.Errorf("Newest = %q with its manager missing; want an error", v)
		}
	})
}
