package inspect

import (
	"bytes"
	"strings"
	"testing"

	"spwn.sh/packages/dependency/tool"
)

// TestRender_HeaderShape verifies the kubectl-describe header lines
// for a minimal agent. Sections after the header render with counts
// and the standard "- " placeholder when a slot is empty.
func TestRender_HeaderShape(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, Model{Agents: []AgentView{{
		Name:    "neo",
		Role:    "worker",
		Runtime: "claude-code",
		World:   "default",
		Status:  StatusRunning,
	}}})

	got := buf.String()
	for _, want := range []string{
		"Name         neo\n",
		"Role         worker\n",
		"Runtime      claude-code\n",
		"World        default   (● running)\n",
		"Dependencies (0 direct, 0 transitive)\n",
		"Skills (0)\n",
		"Hooks (0)\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing line %q in:\n%s", want, got)
		}
	}
}

// TestRender_DepTreeWithDedup checks the cargo-tree shape: nested
// indentation, composition badges aligned at badgeColumn, and the
// second occurrence of a shared dep rendering as "  (*)".
func TestRender_DepTreeWithDedup(t *testing.T) {
	m := Model{Agents: []AgentView{{
		Name:                "neo",
		Role:                "worker",
		Runtime:             "claude-code",
		DirectDepsCount:     2,
		TransitiveDepsCount: 2,
		Deps: []DepNode{
			{Name: "spwn:unix", Version: "24.04"},
			{Name: "spwn:qmd", Version: "1.4.0", Skills: 2, Config: true,
				Children: []DepNode{
					{Name: "spwn:python", Version: "3.12", Kind: tool.KindRuntime,
						Children: []DepNode{
							{Name: "spwn:unix", Version: "24.04", DedupSeen: true},
						}},
				}},
		},
	}}}
	var buf bytes.Buffer
	Render(&buf, m)
	got := buf.String()

	checks := []string{
		"Dependencies (2 direct, 2 transitive)",
		"  spwn:unix@24.04",
		"  spwn:qmd@1.4.0",
		"skills(2) · config",
		"    spwn:python@3.12",
		"runtime",
		"      spwn:unix@24.04  (*)",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// TestRender_DedupSuppressesChildren guards against accidentally
// re-walking a dedup'd subtree: a DepNode with DedupSeen=true
// should render the "(*)" line only, never its Children.
func TestRender_DedupSuppressesChildren(t *testing.T) {
	m := Model{Agents: []AgentView{{
		Name:            "neo",
		Role:            "worker",
		Runtime:         "claude-code",
		DirectDepsCount: 1,
		Deps: []DepNode{
			{Name: "spwn:qmd", DedupSeen: true, Children: []DepNode{
				{Name: "should-not-appear"},
			}},
		},
	}}}
	var buf bytes.Buffer
	Render(&buf, m)
	if strings.Contains(buf.String(), "should-not-appear") {
		t.Errorf("dedup'd subtree leaked children:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "(*)") {
		t.Errorf("dedup marker missing:\n%s", buf.String())
	}
}

// TestRender_SkillsAndHooksAlignOrigin — the two-column layout
// (name + origin, separated by at least 4 spaces) is what makes
// the block scannable. Verify every origin aligns to the same
// column across rows of unequal name length.
func TestRender_SkillsAndHooksAlignOrigin(t *testing.T) {
	m := Model{Agents: []AgentView{{
		Name:    "neo",
		Role:    "worker",
		Runtime: "claude-code",
		Skills: []SkillRef{
			{Name: "a", Origin: "spwn/skills"},
			{Name: "looooooooong-name", Origin: "spwn:qmd"},
		},
		Hooks: []HookRef{
			{Name: "short", Origin: "spwn/hooks"},
			{Name: "loooooonger", Origin: "spwn/hooks"},
		},
	}}}
	var buf bytes.Buffer
	Render(&buf, m)
	got := buf.String()

	// Both origin tokens must appear preceded by 4+ spaces (the gap
	// between the padded name column and the origin column).
	for _, frag := range []string{"    spwn/skills", "    spwn:qmd", "    spwn/hooks"} {
		if !strings.Contains(got, frag) {
			t.Errorf("origin column not aligned, missing %q in:\n%s", frag, got)
		}
	}
}

// TestRender_MultiAgentSeparatedByBlankLine — two agent blocks
// must be separated by exactly one blank line (the helm/terraform
// "whitespace is enough" convention).
func TestRender_MultiAgentSeparatedByBlankLine(t *testing.T) {
	m := Model{Agents: []AgentView{
		{Name: "neo", Role: "worker", Runtime: "claude-code"},
		{Name: "trinity", Role: "chief", Runtime: "codex"},
	}}
	var buf bytes.Buffer
	Render(&buf, m)
	got := buf.String()

	// Find where block 2 starts and confirm a single "\n\n" (one
	// blank line) precedes it — no divider chars, no extra blanks.
	idx := strings.Index(got, "Name         trinity")
	if idx < 0 {
		t.Fatalf("second agent block missing:\n%s", got)
	}
	before := got[:idx]
	if !strings.HasSuffix(before, "\n\n") {
		t.Errorf("block separator should be exactly one blank line, got %q", before[len(before)-6:])
	}
	// Reject three-or-more consecutive newlines anywhere after the first block.
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("found triple newline; want single blank-line separators:\n%s", got)
	}
}

// TestRender_StatusGlyphs verifies every known Status renders with
// the expected glyph; unknown falls back to "○ stopped" (so a live
// lookup that couldn't reach the architect still reads cleanly).
func TestRender_StatusGlyphs(t *testing.T) {
	cases := []struct {
		status Status
		want   string
	}{
		{StatusRunning, "● running"},
		{StatusIdle, "◐ idle"},
		{StatusStopped, "○ stopped"},
		{StatusUnknown, "○ stopped"},
	}
	for _, tc := range cases {
		if got := renderStatus(tc.status); got != tc.want {
			t.Errorf("renderStatus(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// TestRender_ComposeBadges — kind, skills count, and config flag
// each contribute a badge; they concatenate with " · " in a stable
// order (kind, skills, config) regardless of the input order.
func TestRender_ComposeBadges(t *testing.T) {
	cases := []struct {
		name string
		in   DepNode
		want string
	}{
		{"empty tool", DepNode{Kind: tool.KindTool}, ""},
		{"skills only", DepNode{Skills: 3}, "skills(3)"},
		{"config only", DepNode{Config: true}, "config"},
		{"runtime only", DepNode{Kind: tool.KindRuntime}, "runtime"},
		{"all three", DepNode{Kind: tool.KindRuntime, Skills: 2, Config: true},
			"runtime · skills(2) · config"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := composeBadges(tc.in); got != tc.want {
				t.Errorf("composeBadges = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRender_EmptyAgentDoesNotCrash — no deps, no skills, no hooks
// is a legitimate state (a just-scaffolded agent). Section headers
// must still render with their (0) counts and the "- " placeholder.
func TestRender_EmptyAgentDoesNotCrash(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, Model{Agents: []AgentView{{
		Name: "newbie", Role: "worker", Runtime: "claude-code",
	}}})
	got := buf.String()
	for _, want := range []string{
		"Dependencies (0 direct, 0 transitive)\n",
		"Skills (0)\n  -\n",
		"Hooks (0)\n  -\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty-agent placeholder missing %q in:\n%s", want, got)
		}
	}
}
