package machines

import "testing"

func TestACandidateCarriesItsDisk(t *testing.T) {
	if c := CandidateOf(VM{Name: "linux-3", Lifetime: Ephemeral, State: Stopped, Resources: Resources{DiskBytes: 12e9}, SizeGB: 20}); c.DiskBytes != 12e9 {
		t.Errorf("measured: %+v", c)
	}
	if c := CandidateOf(VM{Name: "linux-3", Lifetime: Ephemeral, State: Stopped, SizeGB: 20}); c.DiskBytes != 20e9 {
		t.Errorf("sized: %+v", c)
	}
}

// invariant: SPEC B5.9: a cleanup takes the stopped ephemeral workspaces
// and names the running ones, which stay; a persistent one is neither.
func TestCleanupTakesTheStoppedEphemeralsAndNamesTheRunningOnes(t *testing.T) {
	gone, kept := Cleanup([]VM{
		{Name: "linux-1", Lifetime: Ephemeral, State: Running},
		{Name: "linux-3", Lifetime: Ephemeral, State: Stopped},
		{Name: "api", Lifetime: Persistent, State: Stopped},
		{Name: "infra", Lifetime: AlwaysOn, State: Running},
		{Name: "linux-4", Lifetime: Ephemeral, State: Creating},
	})
	if len(gone) != 1 || gone[0].Name != "linux-3" {
		t.Errorf("gone %v, want linux-3 alone", gone)
	}
	if len(kept) != 1 || kept[0].Name != "linux-1" {
		t.Errorf("kept %v, want linux-1 alone", kept)
	}
}
