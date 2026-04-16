package catalog

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestShippedSlugsMatchEmbed is the load-bearing invariant: every
// slug in shippedSlugs must have an embedded directory with an
// example.yaml sidecar, and vice-versa. Dependency-shaped entries
// (no example.yaml) share the embed FS — they are the other face
// of the catalog and get filtered out here so the gallery list
// stays canonical.
//
// Runs against catalogFS so it exercises the exact bytes that ship
// in the compiled binary - NOT the filesystem.
func TestShippedSlugsMatchEmbed(t *testing.T) {
	entries, err := fs.ReadDir(catalogFS, ".")
	if err != nil {
		t.Fatalf("read embed root: %v", err)
	}

	embeddedExamples := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := fs.Stat(catalogFS, e.Name()+"/example.yaml"); err == nil {
			embeddedExamples[e.Name()] = true
		}
	}

	canonical := make(map[string]bool, len(shippedSlugs))
	for _, s := range shippedSlugs {
		canonical[s] = true
	}

	for slug := range canonical {
		if !embeddedExamples[slug] {
			t.Errorf("shippedSlugs lists %q but no embedded %q/example.yaml found", slug, slug)
		}
	}
	for slug := range embeddedExamples {
		if !canonical[slug] {
			t.Errorf("embedded example %q is missing from shippedSlugs — add it to keep the gallery canonical", slug)
		}
	}
}

// TestShippedSlugsStructure asserts every shipped slug has the
// minimum filesystem contract that Install and Get depend on:
//   <slug>/example.yaml
//   <slug>/README.md
//   <slug>/spwn.yaml
//   <slug>/agents/<at-least-one-dir>/identity/profile.md
//
// Without these, the binary ships but misbehaves at runtime.
func TestShippedSlugsStructure(t *testing.T) {
	for _, slug := range shippedSlugs {
		t.Run(slug, func(t *testing.T) {
			mustExist := []string{
				slug + "/example.yaml",
				slug + "/README.md",
				slug + "/spwn.yaml",
				slug + "/spwn.lock",
			}
			for _, p := range mustExist {
				if _, err := fs.Stat(catalogFS, p); err != nil {
					t.Errorf("missing %s: %v", p, err)
				}
			}

			// At least one agent directory, each with identity/profile.md.
			agentEntries, err := fs.ReadDir(catalogFS, slug+"/agents")
			if err != nil {
				t.Errorf("read %s/agents: %v", slug, err)
				return
			}
			hasAgent := false
			for _, e := range agentEntries {
				if e.IsDir() {
					hasAgent = true
					// Every agent must have identity/profile.md (the current Mind layout).
					profilePath := slug + "/agents/" + e.Name() + "/identity/profile.md"
					if _, err := fs.Stat(catalogFS, profilePath); err != nil {
						t.Errorf("%s: agent %q missing identity/profile.md", slug, e.Name())
					}
					// And an agent.yaml so Install can wire up runtime/tools.
					agentYAML := slug + "/agents/" + e.Name() + "/agent.yaml"
					if _, err := fs.Stat(catalogFS, agentYAML); err != nil {
						t.Errorf("%s: agent %q missing agent.yaml", slug, e.Name())
					}
				}
			}
			if !hasAgent {
				t.Errorf("%s: no agent directory under agents/", slug)
			}
		})
	}
}

// TestList_StartupIsFirst verifies the gallery ordering - startup
// should be the first example users see since it's the best showcase.
func TestList_StartupIsFirst(t *testing.T) {
	got, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("List returned no examples")
	}
	if got[0].Slug != "startup" {
		t.Errorf("first example is %q, want %q", got[0].Slug, "startup")
	}
}

// TestList_ReturnsExactCount locks in the list count so accidentally
// dropping one shows up in CI before it reaches a binary.
func TestList_ReturnsExactCount(t *testing.T) {
	got, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(shippedSlugs) {
		t.Fatalf("List() returned %d examples, want %d", len(got), len(shippedSlugs))
	}
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestList_AllShippedExamplesParse(t *testing.T) {
	got, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	wantSlugs := map[string]bool{
		"macrohard":        false,
		"matrix":           false,
		"paperclip-factory": false,
		"research-lab":     false,
		"startup":          false,
	}
	for _, ex := range got {
		if _, ok := wantSlugs[ex.Slug]; !ok {
			t.Errorf("unexpected slug %q", ex.Slug)
			continue
		}
		wantSlugs[ex.Slug] = true
		if ex.Name == "" || ex.Tagline == "" || ex.Description == "" {
			t.Errorf("%s: missing metadata (name=%q tagline=%q)", ex.Slug, ex.Name, ex.Tagline)
		}
		if len(ex.Agents) == 0 {
			t.Errorf("%s: no agents declared", ex.Slug)
		}
		if len(ex.Worlds) == 0 {
			t.Errorf("%s: no worlds declared", ex.Slug)
		}
	}
	for slug, found := range wantSlugs {
		if !found {
			t.Errorf("example %q missing from List()", slug)
		}
	}
}

func TestGet_IncludesReadme(t *testing.T) {
	ex, err := Get("matrix")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ex.Readme == "" {
		t.Error("README body should be populated by Get")
	}
}

func TestGet_UnknownSlug(t *testing.T) {
	if _, err := Get("nope"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestInstall_CopiesAgentsAndWorldsIdempotently(t *testing.T) {
	base := t.TempDir()

	rep, err := Install("matrix", base)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !rep.ManifestAdded {
		t.Error("expected ManifestAdded to be true on fresh install")
	}
	if len(rep.WorldsAdded) == 0 {
		t.Error("expected at least one world to be added")
	}
	if len(rep.AgentsAdded) == 0 {
		t.Error("expected at least one agent to be added")
	}

	// Files should exist in the new project tree layout.
	if !exists(filepath.Join(base, "spwn.yaml")) {
		t.Error("spwn.yaml was not written")
	}
	if !exists(filepath.Join(base, "spwn", "agents", "neo", "identity", "profile.md")) {
		t.Error("agent identity/profile.md was not copied into spwn/agents/")
	}
	if !exists(filepath.Join(base, "spwn", "agents", "neo", "agent.yaml")) {
		t.Error("agent.yaml was not copied into spwn/agents/")
	}

	// Re-install should be a no-op: everything skipped.
	rep2, err := Install("matrix", base)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if rep2.ManifestAdded {
		t.Error("second install should not re-add the manifest")
	}
	if len(rep2.WorldsAdded) != 0 || len(rep2.AgentsAdded) != 0 {
		t.Errorf("second install should be no-op, got %+v", rep2)
	}
	if len(rep2.WorldsSkipped) == 0 || len(rep2.AgentsSkipped) == 0 {
		t.Errorf("second install should report skips, got %+v", rep2)
	}
}

func TestInstall_PreservesLocalEdits(t *testing.T) {
	base := t.TempDir()
	_, err := Install("paperclip-factory", base)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// User edits the installed manifest.
	manifestPath := filepath.Join(base, "spwn.yaml")
	mine := []byte("version: 2\nname: paperclip-factory\nworlds:\n  paperclip-factory:\n    agents: [clippy]\n    workspaces: [.]\n    tools:\n      - \"@spwn/git\"\n")
	if err := os.WriteFile(manifestPath, mine, 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-install must NOT overwrite their edits.
	_, err = Install("paperclip-factory", base)
	if err != nil {
		t.Fatalf("re-Install: %v", err)
	}
	got, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(mine) {
		t.Errorf("user edits were overwritten: got %q", string(got))
	}
}

func TestInstall_UnknownSlug(t *testing.T) {
	if _, err := Install("nope", t.TempDir()); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
