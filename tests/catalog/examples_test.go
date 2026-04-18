package catalog_test

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/dependency"
)

// Structural embed-walking tests live inside the spwn adapter
// (packages/dependency/internal/adapters/spwn/structure_test.go)
// where they can access the embed FS white-box. This file covers
// the public facade only.

// TestList_StartupIsFirst verifies the gallery ordering — startup
// should be the first example users see since it's the multi-agent
// showcase.
func TestList_StartupIsFirst(t *testing.T) {
	got, err := dependency.Gallery()
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
	got, err := dependency.Gallery()
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
	if _, err := dependency.GalleryEntryBySlug("nope"); err != dependency.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_PureDependencyIsNotGalleryEligible(t *testing.T) {
	// spwn:unix has no worlds: section so Get must treat it as
	// not-gallery-eligible.
	if _, err := dependency.GalleryEntryBySlug("unix"); err != dependency.ErrNotFound {
		t.Errorf("Get(\"unix\") should fail: deps without worlds are not gallery-eligible (got err=%v)", err)
	}
}

func TestInstall_CopiesAgentsAndWorldsIdempotently(t *testing.T) {
	base := t.TempDir()

	rep, err := dependency.Install("matrix", base)
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
	rep2, err := dependency.Install("matrix", base)
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
	_, err := dependency.Install("paperclip-factory", base)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	manifestPath := filepath.Join(base, "spwn.yaml")
	mine := []byte("version: 2\nname: paperclip-factory\nworlds:\n  paperclip-factory:\n    agents: [clippy]\n    workspaces: [.]\n")
	if err := os.WriteFile(manifestPath, mine, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = dependency.Install("paperclip-factory", base)
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
	if _, err := dependency.Install("nope", t.TempDir()); err != dependency.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
