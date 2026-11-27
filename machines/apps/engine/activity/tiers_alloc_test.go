//go:build !race

package activity

import (
	"testing"
	"time"
)

func TestASampleRollsIntoEveryTierWithoutAllocating(t *testing.T) {
	sp := samplerAt(time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC))
	sp.run(time.Hour, time.Second, func(time.Time) float64 { return 20 })
	if allocs := testing.AllocsPerRun(1000, func() { sp.step(time.Second, 20) }); allocs != 0 {
		t.Errorf("a sample's insert allocates %v times; want none once the tiers are made", allocs)
	}
	sp.s.need(T1)
	if allocs := testing.AllocsPerRun(1000, func() { sp.step(time.Second, 20) }); allocs != 0 {
		t.Errorf("a sample's insert allocates %v times once the 10 s tier is made; want none", allocs)
	}
}
