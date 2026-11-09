package main

import (
	"bytes"
	"github.com/jterrazz/studio-station/src/cmd/generate/pages"
	"github.com/jterrazz/studio-station/src/page"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func root(t *testing.T) string {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "..", "..", "..")
}

func TestTheToolbeltProjectionHasNotDriftedFromTheCatalogue(t *testing.T) {
	current, err := os.ReadFile(filepath.Join(root(t), defaultPath))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := render(string(current))
	if err != nil {
		t.Fatal(err)
	}
	if rendered != string(current) {
		t.Fatalf("%s is out of sync with src/catalog: run make generate and commit the result", defaultPath)
	}
}

func TestTheFootprintPageHasNotDriftedFromTheCatalogue(t *testing.T) {
	current, err := os.ReadFile(filepath.Join(root(t), footprintPath))
	if err != nil {
		t.Fatal(err)
	}
	if footprint() != string(current) {
		t.Fatalf("%s is out of sync with src/catalog: run make generate and commit the result", footprintPath)
	}
}

func TestThePageFixturesHaveNotDriftedFromThePageFunctions(t *testing.T) {
	want := pageFixtures()
	dir := filepath.Join(root(t), pagesDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		t.Errorf("%s holds %d fixtures, the page functions build %d: run make generate", pagesDir, len(entries), len(want))
	}
	for name, body := range want {
		current, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(current, body) {
			t.Errorf("%s/%s is out of sync with its page function: run make generate and commit the result", pagesDir, name)
		}
	}
}

func TestTheRecordedStreamHasNotDriftedFromTheDaemon(t *testing.T) {
	current, err := os.ReadFile(filepath.Join(root(t), streamPath))
	if err != nil || !bytes.Equal(current, menubarStream()) {
		t.Errorf("%s is out of sync with the daemon's encoder: run make generate and commit the result", streamPath)
	}
}

func TestTheRecordedEventsHaveNotDriftedFromTheEngine(t *testing.T) {
	current, err := os.ReadFile(filepath.Join(root(t), eventsPath))
	if err != nil || !bytes.Equal(current, pages.EventsStream()) {
		t.Errorf("%s is out of sync with the engine's events: run make generate and commit the result", eventsPath)
	}
}

func TestThePlayWorldHasNotDriftedFromTheStoriesMachines(t *testing.T) {
	base, err := os.ReadFile(filepath.Join(root(t), pages.PlayWorldBase))
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(filepath.Join(root(t), pages.PlayWorldPath))
	if err != nil || !bytes.Equal(current, pages.PlayWorld(base)) {
		t.Errorf("%s is out of sync with the stories' machines: run make generate and commit the result", pages.PlayWorldPath)
	}
}

func TestTheStoryPagesHaveNotDriftedFromTheirWorlds(t *testing.T) {
	want := storyPages()
	dir := filepath.Join(root(t), storiesDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if _, ok := want[e.Name()]; !ok {
			t.Errorf("%s/%s is drawn by no story world: delete it, then run make generate", storiesDir, e.Name())
		}
	}
	for name, body := range want {
		current, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(current, body) {
			t.Errorf("%s/%s is out of sync with its world: run make generate and commit the result", storiesDir, name)
		}
	}
}

func TestEveryStoryPagePassesTheChecksOfSeal(t *testing.T) {
	pages.Levels = levels
	for _, f := range pages.StoryPages() {
		if err := page.Check(f.Page); err != nil {
			t.Errorf("%s: %v", f.Name, err)
		}
	}
}

func TestEveryPageAWorldNamesIsDrawn(t *testing.T) {
	drawn := storyPages()
	for _, w := range pages.Worlds() {
		for _, p := range w.Pages {
			if _, ok := drawn[w.Name+"."+p+".json"]; !ok {
				t.Errorf("the world %s names the page %s, which its scene does not draw", w.Name, p)
			}
		}
	}
}

func TestEveryVariantIsOneEditOfTheMachinesWorld(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range pages.Worlds() {
		if seen[w.Name] {
			t.Errorf("%s: two worlds share a name", w.Name)
		}
		seen[w.Name] = true
		if w.Name != "machines" && (w.Edit == nil || !strings.HasPrefix(w.Name, "machines.") || len(w.Pages) == 0) {
			t.Errorf("%s: a variant is named machines.<name>, edits the machines world once, and names the pages it draws", w.Name)
		}
	}
}
