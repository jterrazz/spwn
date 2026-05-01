package architect

import (
	"reflect"
	"testing"
)

func TestFacultiesForRuntime_AppendsLocalSkills(t *testing.T) {
	got := facultiesForRuntime(
		[]string{"spwn:unix", "spwn:git"},
		[]string{"skill/zeta", "tool/local", "skill/focus", "skill/focus"},
	)
	want := []string{"spwn:unix", "spwn:git", "skill/focus", "skill/zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("facultiesForRuntime() = %#v, want %#v", got, want)
	}
}

func TestFacultiesForRuntime_DedupesVerifiedSkill(t *testing.T) {
	got := facultiesForRuntime([]string{"spwn:unix", "skill/focus"}, []string{"skill/focus"})
	want := []string{"spwn:unix", "skill/focus"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("facultiesForRuntime() = %#v, want %#v", got, want)
	}
}

func TestRuntimePromptFile(t *testing.T) {
	tests := map[string]string{
		"":            "CLAUDE.md",
		"claude-code": "CLAUDE.md",
		"codex":       "AGENTS.md",
		"other":       "prompt",
	}
	for runtimeName, want := range tests {
		if got := runtimePromptFile(runtimeName); got != want {
			t.Fatalf("runtimePromptFile(%q) = %q, want %q", runtimeName, got, want)
		}
	}
}
