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
