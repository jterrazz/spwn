package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// invariant: a page ends with one or two next steps, never more (SPEC §5,
// vision 07 "on finit par une ou deux suites").
const mostSteps = 2

func TestAPageEndsWithAtMostTwoNextSteps(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "apps", "engine", "ui", "testdata", "pages")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var p struct {
			Hints []struct {
				Command string `json:"command"`
			} `json:"hints"`
		}
		if err := json.Unmarshal(body, &p); err != nil {
			t.Fatal(err)
		}
		if len(p.Hints) > mostSteps {
			t.Errorf("apps/engine/ui/testdata/pages/%s ends with %d next steps, more than %d", e.Name(), len(p.Hints), mostSteps)
		}
	}
}
