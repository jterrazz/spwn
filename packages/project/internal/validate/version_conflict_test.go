package validate

import (
	"testing"

	intmanifest "spwn.sh/packages/project/internal/manifest"
)

// TestRulePackageVersionConflict_flagsMismatchAcrossAgents locks in
// the invariant that two agents sharing a world cannot pin the same
// spwn: package at different versions. The image builder can only
// install one version of a given package per world, so "A wants
// 24.04, B wants 22.04" is ambiguous and must fail fast.
func TestRulePackageVersionConflict_flagsMismatchAcrossAgents(t *testing.T) {
	root := t.TempDir()

	neo := scaffoldAgent(t, root, "neo", `name: neo
dependencies:
  - "spwn:unix@24.04"
`)
	morpheus := scaffoldAgent(t, root, "morpheus", `name: morpheus
dependencies:
  - "spwn:unix@22.04"
`)

	in := Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "t",
			Worlds: map[string]intmanifest.World{
				"main": {Agents: []string{"neo", "morpheus"}, Workspaces: []string{"."}},
			},
		},
		AgentRefs: []AgentRef{neo, morpheus},
	}

	issues := rulePackVersionConflict(in)
	if len(issues) != 1 {
		t.Fatalf("want 1 conflict issue, got %d: %+v", len(issues), issues)
	}
	if !contains(issues[0].Message, "spwn:unix") {
		t.Errorf("message should mention spwn:unix: %q", issues[0].Message)
	}
	if !contains(issues[0].Message, "conflicting versions") {
		t.Errorf("message should mention conflict: %q", issues[0].Message)
	}
}

// TestRulePackageVersionConflict_identicalVersionsOK ensures the
// rule doesn't false-positive when every agent agrees on the version.
func TestRulePackageVersionConflict_identicalVersionsOK(t *testing.T) {
	root := t.TempDir()

	neo := scaffoldAgent(t, root, "neo", `name: neo
dependencies:
  - "spwn:unix@24.04"
`)
	morpheus := scaffoldAgent(t, root, "morpheus", `name: morpheus
dependencies:
  - "spwn:unix@24.04"
`)

	in := Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "t",
			Worlds: map[string]intmanifest.World{
				"main": {Agents: []string{"neo", "morpheus"}, Workspaces: []string{"."}},
			},
		},
		AgentRefs: []AgentRef{neo, morpheus},
	}

	if got := rulePackVersionConflict(in); len(got) != 0 {
		t.Errorf("matching versions should not flag, got %+v", got)
	}
}

// TestRulePackageVersionConflict_singleAgentSkipped ensures the rule
// is a no-op for single-agent worlds (conflict requires at least two).
func TestRulePackageVersionConflict_singleAgentSkipped(t *testing.T) {
	root := t.TempDir()

	neo := scaffoldAgent(t, root, "neo", `name: neo
dependencies:
  - "spwn:unix@24.04"
`)

	in := Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "t",
			Worlds: map[string]intmanifest.World{
				"solo": {Agents: []string{"neo"}, Workspaces: []string{"."}},
			},
		},
		AgentRefs: []AgentRef{neo},
	}

	if got := rulePackVersionConflict(in); len(got) != 0 {
		t.Errorf("single-agent world should not fire, got %+v", got)
	}
}
