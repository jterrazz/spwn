package api

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"spwn.sh/packages/platform"
)

type galleryCard struct {
	Agents  []string `json:"agents"`
	Command string   `json:"command"`
	Name    string   `json:"name"`
	Slug    string   `json:"slug"`
	Worlds  []string `json:"worlds"`
}

func TestGallery_DescribesTheStartupExample(t *testing.T) {
	_, mux := newTestServer(t)

	w := doJSON(t, mux, "GET", "/api/examples/startup", nil)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var card galleryCard
	if err := json.NewDecoder(w.Body).Decode(&card); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := galleryCard{
		Agents:  []string{"analyst", "ceo", "devops"},
		Command: "spwn up startup",
		Name:    "Startup",
		Slug:    "startup",
		Worlds:  []string{"startup"},
	}
	if !reflect.DeepEqual(card, want) {
		t.Errorf("startup card = %+v, want %+v", card, want)
	}
}

func TestGallery_OffersOneNamedCardPerExample(t *testing.T) {
	_, mux := newTestServer(t)

	w := doJSON(t, mux, "GET", "/api/examples", nil)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var gallery struct {
		Examples []galleryCard `json:"examples"`
	}
	if err := json.NewDecoder(w.Body).Decode(&gallery); err != nil {
		t.Fatalf("decode: %v", err)
	}
	slugs := make([]string, 0, len(gallery.Examples))
	for _, example := range gallery.Examples {
		if example.Name == "" {
			t.Errorf("example %q carries no name", example.Slug)
		}
		slugs = append(slugs, example.Slug)
	}
	if len(slugs) == 0 || slugs[0] != "startup" || !slices.Contains(slugs, "matrix") {
		t.Errorf("gallery slugs = %v, want startup first and matrix among them", slugs)
	}
}

func TestGallery_UnknownExampleIsNotFound(t *testing.T) {
	_, mux := newTestServer(t)

	w := doJSON(t, mux, "GET", "/api/examples/no-such-example", nil)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGallery_InstallLandsTheExampleAgents(t *testing.T) {
	_, mux := newTestServer(t)
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "spwn.yaml"), "version: 1\nname: gallery\n")
	platform.SetProjectRoot(project)
	t.Cleanup(func() { platform.SetProjectRoot("") })

	install := doJSON(t, mux, "POST", "/api/examples/startup/install", nil)
	if install.Code != 200 {
		t.Fatalf("install: expected 200, got %d: %s", install.Code, install.Body.String())
	}
	w := doJSON(t, mux, "GET", "/api/agents", nil)

	var agents []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("decode: %v (body: %s)", err, w.Body.String())
	}
	names := make([]string, 0, len(agents))
	for _, agent := range agents {
		names = append(names, agent.Name)
	}
	slices.Sort(names)
	if !reflect.DeepEqual(names, []string{"analyst", "ceo", "devops"}) {
		t.Errorf("agents after install = %v, want [analyst ceo devops]", names)
	}
}
