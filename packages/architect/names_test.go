package architect

import "testing"

func TestSpawnRefusesANameThatIsNotOneName(t *testing.T) {
	for _, opts := range []SpawnOpts{
		{ConfigName: "../x"},
		{ConfigName: "neo", AgentName: "../../x"},
		{ConfigName: "neo", Agents: []AgentSpec{{Name: "neo"}, {Name: "a/b"}}},
		{ConfigName: "neo", AgentName: ".."},
	} {
		if err := opts.Validate(); err == nil {
			t.Errorf("Validate took %+v", opts)
		}
	}
	if err := (&SpawnOpts{ConfigName: "neo", AgentName: "neo"}).Validate(); err != nil {
		t.Errorf("Validate refused an honest spawn: %v", err)
	}
}
