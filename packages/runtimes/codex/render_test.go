package codex

import (
	"strings"
	"testing"

	"spwn.sh/packages/transpile"
)

// Renderer contract tests. See golden fixtures under
// packages/runtimes/testdata/*/output_codex/ for whole-tree
// diffing; these cases exercise individual invariants at the
// section level so a failing test points at a specific section
// rather than "goldens diverged".

func TestRenderer_Name(t *testing.T) {
	if got := Renderer.Name(); got != "codex" {
		t.Errorf("Name() = %q, want codex", got)
	}
}

func TestRender_EmitsAgentsMDAndRoleMD(t *testing.T) {
	tree, err := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo", Role: "worker"}},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	paths := tree.Paths()
	wantPaths := map[string]bool{
		"agents/neo/AGENTS.md":              false,
		"agents/neo/worlds/home/role.md":    false,
	}
	for _, p := range paths {
		if _, ok := wantPaths[p]; ok {
			wantPaths[p] = true
		}
	}
	for p, seen := range wantPaths {
		if !seen {
			t.Errorf("tree missing %s", p)
		}
	}
}

func TestRender_RoleMDContent(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo", Role: "chief"}},
	})
	got, _ := tree.Get("agents/neo/worlds/home/role.md")
	want := "# Role in home\n\nchief\n"
	if string(got) != want {
		t.Errorf("role.md:\n got %q\nwant %q", got, want)
	}
}

func TestRender_EmptyRoleDefaultsToWorker(t *testing.T) {
	tree, _ := Renderer.Render(transpile.Input{
		WorldID: "home",
		Agents:  []transpile.AgentInput{{Name: "neo"}},
	})
	got, _ := tree.Get("agents/neo/worlds/home/role.md")
	if !strings.Contains(string(got), "worker") {
		t.Errorf("expected default role 'worker' in role.md, got %q", got)
	}
}

func TestGenerateAgentAgentsMD_IncludesAllSections(t *testing.T) {
	out := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo",
		Role:      "worker",
		WorldID:   "home",
		Soul:      []byte("# Neo\n\nYou are relentless."),
		Physics:   "# Physics of This World\n\n## Laws\n- Network: bridge\n",
		Faculties: "# Faculties\n\n## Tools\nspwn:unix\n",
		Roster:    "# Roster - home\n\nsome roster body\n",
		Playbooks: []transpile.PlaybookEntry{{Name: "deploy", Description: "Ship the change."}},
		AgentMD:   []byte("You help with CI audits."),
		KnowledgeMounted: true,
	})
	for _, want := range []string{
		"# neo — worker in world \"home\"",
		"## Identity",
		"You are relentless.",
		"## Physics",
		"Network: bridge",
		"## Faculties",
		"spwn:unix",
		"## Roster",
		"some roster body",
		"## Role here",
		"deployed as a worker in home",
		"## Your playbooks",
		"**deploy** — Ship the change.",
		"## Conventions",
		"World knowledge",
		"## Task",
		"You help with CI audits.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
}

func TestGenerateAgentAgentsMD_NoSoulSkipsIdentity(t *testing.T) {
	out := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo", Role: "worker", WorldID: "home",
		Physics: "# P\n", Faculties: "# F\n", Roster: "# R\n",
	})
	if strings.Contains(out, "## Identity") {
		t.Errorf("blank Soul should omit the Identity section; got:\n%s", out)
	}
}

func TestGenerateAgentAgentsMD_WhitespaceOnlySoulSkipsIdentity(t *testing.T) {
	out := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo", Role: "worker", WorldID: "home",
		Soul:    []byte("   \n\t\n"),
		Physics: "# P\n", Faculties: "# F\n", Roster: "# R\n",
	})
	if strings.Contains(out, "## Identity") {
		t.Errorf("whitespace-only Soul should omit Identity; got:\n%s", out)
	}
}

func TestGenerateAgentAgentsMD_EmptyPlaybooksSkipsSection(t *testing.T) {
	out := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo", Role: "worker", WorldID: "home",
		Physics: "# P\n", Faculties: "# F\n", Roster: "# R\n",
	})
	if strings.Contains(out, "## Your playbooks") {
		t.Errorf("empty Playbooks should omit the section; got:\n%s", out)
	}
}

func TestGenerateAgentAgentsMD_KnowledgeFlagTogglesConvention(t *testing.T) {
	with := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo", Role: "worker", WorldID: "home",
		Physics: "# P\n", Faculties: "# F\n", Roster: "# R\n",
		KnowledgeMounted: true,
	})
	without := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo", Role: "worker", WorldID: "home",
		Physics: "# P\n", Faculties: "# F\n", Roster: "# R\n",
	})
	if !strings.Contains(with, "World knowledge") {
		t.Error("KnowledgeMounted=true should emit the World knowledge bullet")
	}
	if strings.Contains(without, "World knowledge") {
		t.Error("KnowledgeMounted=false should NOT mention World knowledge")
	}
}

func TestGenerateAgentAgentsMD_NoAgentMDSkipsTask(t *testing.T) {
	out := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo", Role: "worker", WorldID: "home",
		Physics: "# P\n", Faculties: "# F\n", Roster: "# R\n",
	})
	if strings.Contains(out, "## Task") {
		t.Errorf("blank AgentMD should omit the Task section; got:\n%s", out)
	}
}

func TestGenerateAgentAgentsMD_HeadingsInInlinedBlocksAreDemoted(t *testing.T) {
	// A physics block has `## Laws` at H2. After inlining under the
	// wrapper's own `## Physics`, those inner H2s must demote to H3
	// so the outline reads cleanly.
	out := GenerateAgentAgentsMD(AgentAgentsMDInput{
		AgentName: "neo", Role: "worker", WorldID: "home",
		Physics:   "# Physics of This World\n\n## Laws\nrules\n",
		Faculties: "# F\n", Roster: "# R\n",
	})
	if !strings.Contains(out, "### Laws") {
		t.Errorf("inner ## Laws should demote to ### Laws; got:\n%s", out)
	}
	if strings.Contains(out, "# Physics of This World") {
		t.Errorf("the inner H1 should be stripped; got:\n%s", out)
	}
}

func TestRender_MultiAgentRosterSeenByEach(t *testing.T) {
	// Both agents should see the same roster; the worldbook helper
	// produces one roster body and every AGENTS.md inlines it.
	tree, err := Renderer.Render(transpile.Input{
		WorldID: "colony",
		Agents: []transpile.AgentInput{
			{Name: "alice", Role: "chief"},
			{Name: "bob", Role: "worker"},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, name := range []string{"alice", "bob"} {
		body, ok := tree.Get("agents/" + name + "/AGENTS.md")
		if !ok {
			t.Fatalf("missing AGENTS.md for %s", name)
		}
		s := string(body)
		if !strings.Contains(s, "**alice** (chief)") || !strings.Contains(s, "**bob** (worker)") {
			t.Errorf("%s's AGENTS.md missing peer roster entries", name)
		}
	}
}
