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

// invariant: SPEC B5.9: the note names the agents still running, never a
// kept VM whose agent finished.
func TestCleanupNamesOnlyTheAgentsStillRunning(t *testing.T) {
	_, kept := Cleanup([]VM{
		{Name: "review-bot", Lifetime: Permanent, State: Running, Agent: &AgentView{State: "waiting"}},
		{Name: "infra", Lifetime: Permanent, State: Running, Agent: &AgentView{State: AgentFinished}},
		{Name: "linux-1", Lifetime: Permanent, State: Running},
	})
	if len(kept) != 1 || kept[0].Name != "review-bot" {
		t.Errorf("kept %v, want review-bot alone", kept)
	}
}
