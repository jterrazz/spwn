package machines

import (
	"testing"
	"time"
)

func TestACandidateCarriesItsIdleTimeAndItsDisk(t *testing.T) {
	now := time.Date(2026, 11, 7, 14, 10, 0, 0, time.UTC)
	done := VM{Name: "agent-7f", Lifetime: Throwaway, State: Running, Resources: Resources{DiskBytes: 12e9},
		Agent: &AgentView{State: AgentFinished, Ended: now.Add(-12 * time.Minute), Reason: "tests pass"}}
	c := CandidateOf(done, now)
	if c.Stopped || c.Idle != 12*time.Minute || c.Reason != "tests pass" || c.DiskBytes != 12e9 {
		t.Errorf("finished: %+v", c)
	}
	stopped := CandidateOf(VM{Name: "linux-3", Lifetime: Throwaway, State: Stopped, SizeGB: 20}, now)
	if !stopped.Stopped || stopped.Idle != 0 || stopped.DiskBytes != 20e9 {
		t.Errorf("stopped: %+v", stopped)
	}
}
