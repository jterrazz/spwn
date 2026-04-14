package base

import "testing"

func TestMindLayers_Count(t *testing.T) {
	if got := len(MindLayers); got != 5 {
		t.Errorf("len(MindLayers) = %d, want 5", got)
	}
}

func TestMindLayers_Contents(t *testing.T) {
	expected := []string{"core", "skills", "knowledge", "playbooks", "journal"}
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
	if WorldImage == "" {
		t.Error("WorldImage is empty")
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
	if SkillsSubDir == "" {
		t.Error("SkillsSubDir is empty")
	}
}
