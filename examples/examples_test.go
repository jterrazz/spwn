package examples

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestShippedSlugsMatchEmbed is the load-bearing invariant: it asserts
// that the canonical shippedSlugs list, the go:embed directive, and the
// on-disk template directories all agree. If any of the three drift
// (someone adds a directory without updating the embed, or updates
// shippedSlugs but forgets the embed, etc.), this test fails loudly.
//
// Runs against templatesFS so it exercises the exact bytes that ship
// in the compiled binary - NOT the filesystem.
func TestShippedSlugsMatchEmbed(t *testing.T) {
	entries, err := fs.ReadDir(templatesFS, ".")
	if err != nil {
		t.Fatalf("read embed root: %v", err)
	}

	embedded := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			embedded = append(embedded, e.Name())
		}
	}
	sort.Strings(embedded)

	canonical := append([]string(nil), shippedSlugs...)
	sort.Strings(canonical)

	if !stringsEqual(embedded, canonical) {
		t.Fatalf("shippedSlugs %v != embedded dirs %v - update the go:embed directive AND shippedSlugs together when adding a template", canonical, embedded)
	}
}

// TestShippedSlugsStructure asserts every shipped slug has the
// minimum filesystem contract that Install and Get depend on:
//   <slug>/example.yaml
//   <slug>/README.md
//   <slug>/agents/<at-least-one-dir>/
//   <slug>/worlds/<at-least-one-yaml>
//
// Without these, the binary ships but misbehaves at runtime.
func TestShippedSlugsStructure(t *testing.T) {
	for _, slug := range shippedSlugs {
		t.Run(slug, func(t *testing.T) {
			mustExist := []string{
				slug + "/example.yaml",
				slug + "/README.md",
			}
			for _, p := range mustExist {
				if _, err := fs.Stat(templatesFS, p); err != nil {
					t.Errorf("missing %s: %v", p, err)
				}
			}

			// At least one agent directory, each with core/profile.md.
			agentEntries, err := fs.ReadDir(templatesFS, slug+"/agents")
			if err != nil {
				t.Errorf("read %s/agents: %v", slug, err)
				return
			}
			hasAgent := false
			for _, e := range agentEntries {
				if e.IsDir() {
					hasAgent = true
					// Every agent must have core/profile.md (the current Mind layout).
					profilePath := slug + "/agents/" + e.Name() + "/core/profile.md"
					if _, err := fs.Stat(templatesFS, profilePath); err != nil {
						t.Errorf("%s: agent %q missing core/profile.md", slug, e.Name())
					}
				}
			}
			if !hasAgent {
				t.Errorf("%s: no agent directory under agents/", slug)
			}

			// At least one world yaml.
			worldEntries, err := fs.ReadDir(templatesFS, slug+"/worlds")
			if err != nil {
				t.Errorf("read %s/worlds: %v", slug, err)
				return
			}
			hasWorld := false
			for _, e := range worldEntries {
				if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
					hasWorld = true
					break
				}
			}
			if !hasWorld {
				t.Errorf("%s: no *.yaml under worlds/", slug)
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

func TestList_AllShippedTemplatesParse(t *testing.T) {
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
			t.Errorf("template %q missing from List()", slug)
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
	if len(rep.WorldsAdded) == 0 {
		t.Error("expected at least one world to be added")
	}
	if len(rep.AgentsAdded) == 0 {
		t.Error("expected at least one agent to be added")
	}

	// Files should exist.
	if !exists(filepath.Join(base, "worlds", "matrix.yaml")) {
		t.Error("matrix.yaml was not written")
	}
	if !exists(filepath.Join(base, "agents", "neo", "core", "profile.md")) {
		t.Error("agent core/profile.md was not copied")
	}

	// Re-install should be a no-op: everything skipped.
	rep2, err := Install("matrix", base)
	if err != nil {
		t.Fatalf("second Install: %v", err)
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

	// User edits the installed world yaml.
	worldYAML := filepath.Join(base, "worlds", "paperclip-factory.yaml")
	mine := []byte("# user edits\nphysics:\n  constants:\n    cpu: 99\n")
	if err := os.WriteFile(worldYAML, mine, 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-install must NOT overwrite their edits.
	_, err = Install("paperclip-factory", base)
	if err != nil {
		t.Fatalf("re-Install: %v", err)
	}
	got, err := os.ReadFile(worldYAML)
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
