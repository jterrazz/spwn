package foundation

import "testing"

func TestMindLayers_Count(t *testing.T) {
	if got := len(MindLayers); got != 6 {
		t.Errorf("len(MindLayers) = %d, want 6", got)
	}
}

func TestMindLayers_Contents(t *testing.T) {
	expected := []string{"personas", "skills", "knowledge", "playbooks", "journal", "sessions"}
	for _, want := range expected {
		found := false
		for _, layer := range MindLayers {
			if layer == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("MindLayers missing expected layer %q", want)
		}
	}
}

func TestDefaultConstants_Sanity(t *testing.T) {
	if DefaultCPU <= 0 {
		t.Errorf("DefaultCPU = %d, want > 0", DefaultCPU)
	}
	if DefaultMemory == "" {
		t.Error("DefaultMemory is empty")
	}
	if DefaultDisk == "" {
		t.Error("DefaultDisk is empty")
	}
	if DefaultTimeout == "" {
		t.Error("DefaultTimeout is empty")
	}
	if DefaultBackend == "" {
		t.Error("DefaultBackend is empty")
	}
	if BaseImage == "" {
		t.Error("BaseImage is empty")
	}
	if DefaultMaxProcs <= 0 {
		t.Errorf("DefaultMaxProcs = %d, want > 0", DefaultMaxProcs)
	}
}

func TestDirectoryConstants_NonEmpty(t *testing.T) {
	if SpwnBaseDir == "" {
		t.Error("SpwnBaseDir is empty")
	}
	if WorldsSubDir == "" {
		t.Error("WorldsSubDir is empty")
	}
	if AgentsSubDir == "" {
		t.Error("AgentsSubDir is empty")
	}
	if StateFileName == "" {
		t.Error("StateFileName is empty")
	}
	if OrgFileName == "" {
		t.Error("OrgFileName is empty")
	}
	if ClawStateFileName == "" {
		t.Error("ClawStateFileName is empty")
	}
	if SkillsSubDir == "" {
		t.Error("SkillsSubDir is empty")
	}
	if ClawSubDir == "" {
		t.Error("ClawSubDir is empty")
	}
}
