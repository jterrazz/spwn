package spwn_cli

import (
	"io/fs"
	"testing"
)

func TestSpwn_Name(t *testing.T) {
	if Tool.Name() != "@spwn" {
		t.Errorf("expected @spwn, got %s", Tool.Name())
	}
}

func TestSpwn_HasSkills(t *testing.T) {
	s := Tool.Skills()
	if s == nil {
		t.Fatal("expected skills FS")
	}
	_, err := fs.ReadFile(s, "SKILL.md")
	if err != nil {
		t.Errorf("expected SKILL.md: %v", err)
	}
}
