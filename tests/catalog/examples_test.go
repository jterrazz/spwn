package catalog_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/catalog"
	"spwn.sh/packages/dependency"
)

// hasWorlds reports whether a parsed catalog schema declares at
// least one world. Mirrors the internal helper in catalog/ — kept
// local here so the test package doesn't depend on that predicate
// being exported.
func hasWorlds(s *dependency.Schema) bool {
	return s != nil && len(s.Worlds.Content) > 0
}

// TestShippedSlugsMatchEmbed asserts every gallery-eligible entry
// (one with a `worlds:` section in spwn.yaml) is reachable via
// ShippedSlugs(), and vice-versa. Dependency-shaped entries (no
// worlds:) live in the same embed FS but stay out of the gallery.
//
// Runs against catalog.EmbedFS() so it exercises the exact bytes
// that ship in the compiled binary — not the filesystem.
func TestShippedSlugsMatchEmbed(t *testing.T) {
	embed := catalog.EmbedFS()
	entries, err := fs.ReadDir(embed, ".")
	if err != nil {
		t.Fatalf("read embed root: %v", err)
	}

	gallerySlugs := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		schema, err := catalog.EntrySchema(e.Name())
		if err != nil {
			continue
		}
		if hasWorlds(schema) {
			gallerySlugs[e.Name()] = true
		}
	}

	canonical := make(map[string]bool)
	for _, s := range catalog.ShippedSlugs() {
		canonical[s] = true
	}

	for slug := range canonical {
		if !gallerySlugs[slug] {
			t.Errorf("ShippedSlugs lists %q but its spwn.yaml has no worlds: section", slug)
		}
	}
	for slug := range gallerySlugs {
		if !canonical[slug] {
			t.Errorf("embedded gallery entry %q is missing from ShippedSlugs", slug)
		}
	}
}

// TestShippedSlugsStructure asserts every gallery entry ships the
// minimum filesystem contract that Install and Get depend on:
//
//	<slug>/spwn.yaml
//	<slug>/spwn.lock
//	<slug>/agents/<at-least-one-dir>/SOUL.md
//	<slug>/agents/<at-least-one-dir>/agent.yaml
//
// Without these, the binary ships but misbehaves at runtime.
func TestShippedSlugsStructure(t *testing.T) {
	embed := catalog.EmbedFS()
	for _, slug := range catalog.ShippedSlugs() {
		t.Run(slug, func(t *testing.T) {
			for _, p := range []string{slug + "/spwn.yaml", slug + "/spwn.lock"} {
				if _, err := fs.Stat(embed, p); err != nil {
					t.Errorf("missing %s: %v", p, err)
				}
			}

			agentEntries, err := fs.ReadDir(embed, slug+"/agents")
			if err != nil {
				t.Errorf("read %s/agents: %v", slug, err)
				return
			}
			hasAgent := false
			for _, e := range agentEntries {
				if !e.IsDir() {
					continue
				}
				hasAgent = true
				profilePath := slug + "/agents/" + e.Name() + "/SOUL.md"
				if _, err := fs.Stat(embed, profilePath); err != nil {
					t.Errorf("%s: agent %q missing SOUL.md", slug, e.Name())
				}
				agentYAML := slug + "/agents/" + e.Name() + "/agent.yaml"
				if _, err := fs.Stat(embed, agentYAML); err != nil {
					t.Errorf("%s: agent %q missing agent.yaml", slug, e.Name())
				}
			}
			if !hasAgent {
				t.Errorf("%s: no agent directory under agents/", slug)
			}
		})
	}
}

// TestList_StartupIsFirst verifies the gallery ordering — startup
// should be the first example users see since it's the multi-agent
// showcase.
func TestList_StartupIsFirst(t *testing.T) {
	got, err := catalog.List()
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

func TestList_AllShippedExamplesParse(t *testing.T) {
	got, err := catalog.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	wantSlugs := map[string]bool{
		"macrohard":         false,
		"matrix":            false,
		"paperclip-factory": false,
		"research-lab":      false,
		"severance":         false,
		"startup":           false,
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

func TestGet_UnknownSlug(t *testing.T) {
	if _, err := catalog.Get("nope"); err != catalog.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_PureDependencyIsNotGalleryEligible(t *testing.T) {
	// spwn:unix has no worlds: section so Get must treat it as
	// not-gallery-eligible.
	if _, err := catalog.Get("unix"); err != catalog.ErrNotFound {
		t.Errorf("Get(\"unix\") should fail: deps without worlds are not gallery-eligible (got err=%v)", err)
	}
}

func TestInstall_CopiesAgentsAndWorldsIdempotently(t *testing.T) {
	base := t.TempDir()

	rep, err := catalog.Install("matrix", base)
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

	if !fileExists(filepath.Join(base, "spwn.yaml")) {
		t.Error("spwn.yaml was not written")
	}
	if !fileExists(filepath.Join(base, "spwn", "agents", "neo", "SOUL.md")) {
		t.Error("agent SOUL.md was not copied into spwn/agents/")
	}
	if !fileExists(filepath.Join(base, "spwn", "agents", "neo", "agent.yaml")) {
		t.Error("agent.yaml was not copied into spwn/agents/")
	}

	// Re-install: no new additions, every world + agent skipped.
	rep2, err := catalog.Install("matrix", base)
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
	_, err := catalog.Install("paperclip-factory", base)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	manifestPath := filepath.Join(base, "spwn.yaml")
	mine := []byte("version: 2\nname: paperclip-factory\nworlds:\n  paperclip-factory:\n    agents: [clippy]\n    workspaces: [.]\n")
	if err := os.WriteFile(manifestPath, mine, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = catalog.Install("paperclip-factory", base)
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
	if _, err := catalog.Install("nope", t.TempDir()); err != catalog.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
