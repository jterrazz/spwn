package platform

import "testing"

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
	if SkillsSubDir == "" {
		t.Error("SkillsSubDir is empty")
	}
}
