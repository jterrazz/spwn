package architecture_test

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

type turboTasks struct {
	Tasks map[string]struct {
		Cache  *bool    `json:"cache"`
		Inputs []string `json:"inputs"`
	} `json:"tasks"`
}

func readTurbo(t *testing.T, rel string) turboTasks {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatal(err)
	}
	var out turboTasks
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return out
}

func coveredBy(file string, inputs []string) bool {
	for _, in := range inputs {
		glob, ok := strings.CutPrefix(in, "$TURBO_ROOT$/")
		if !ok {
			continue
		}
		if dir, whole := strings.CutSuffix(glob, "/**"); whole && strings.HasPrefix(file, dir+"/") {
			return true
		}
		if matched, _ := path.Match(glob, file); matched {
			return true
		}
	}
	return false
}

// invariant: Turborepo replays no stale pass. The Go specs read the whole
// repository (the manual, the catalog, the menu bar's story pages), which no
// input list of theirs holds, so their run is never cached; the engine's
// tests read files outside apps/engine, each an input of its test task.
func TestTurboReplaysNoStalePass(t *testing.T) {
	if c := readTurbo(t, "turbo.json").Tasks["specs#test"].Cache; c == nil || *c {
		t.Errorf("turbo.json caches specs#test: a pass of the Go specs would be replayed after a docs or catalog edit")
	}
	inputs := readTurbo(t, "apps/engine/turbo.json").Tasks["test"].Inputs
	for _, file := range []string{
		"catalog/packages/command-line/jq.yaml",
		"skills/studio-toolbelt/SKILL.md",
		"docs/reference/footprint.md",
		"packages/api/schemas/station.machinefile.v1.json",
		"apps/menubar/stories/pages/home.json",
		"specs/usecases/worlds/stories.json",
	} {
		if !coveredBy(file, inputs) {
			t.Errorf("apps/engine/turbo.json's test inputs leave out %s, which the engine's tests read", file)
		}
	}
}
