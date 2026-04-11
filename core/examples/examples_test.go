package examples

import (
	"os"
	"path/filepath"
	"testing"
)

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
	if !exists(filepath.Join(base, "agents", "neo", "identity", "persona.md")) {
		t.Error("agent identity was not copied")
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
