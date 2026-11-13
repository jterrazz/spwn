package architecture_test

import (
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: a JSON fixture the repository commits: a page fixture, a recorded
// stream, a golden file, a spec fixture, calls.json; the worlds are where
// the secrets live, so they are no fixture.
func jsonFixtures(t *testing.T, root string) map[string]string {
	t.Helper()
	var paths []string
	worlds := filepath.Join(root, "specs", "usecases", "worlds")
	for _, top := range []string{"apps/engine", "specs", "apps"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == worlds || d.Name() == "node_modules" || d.Name() == ".build" {
					return filepath.SkipDir
				}
				return nil
			}
			if ext := filepath.Ext(path); ext == ".json" || ext == ".jsonl" {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	out := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, path)
		out[filepath.ToSlash(rel)] = string(data)
	}
	return out
}

// value: every secret a world holds: what station keeps in a Mac's vault
// and what another tool keeps there, by value.
func worldSecrets(t *testing.T, root string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "specs", "usecases", "worlds", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			Macs map[string]struct {
				Vault   map[string][]byte `json:"vault"`
				Secrets map[string]string `json:"secrets"`
			} `json:"macs"`
		}
		if err := json.Unmarshal(data, &w); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, mac := range w.Macs {
			for _, v := range mac.Vault {
				out = append(out, string(v))
			}
			for _, v := range mac.Secrets {
				out = append(out, v)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(slices.DeleteFunc(out, func(s string) bool { return strings.TrimSpace(s) == "" }))
}

// value: the token shapes of catalog.LeakShapes that name their issuer
// (GitHub, Anthropic, npm, Slack, a JWT), without the bare run of base64
// a digest also is.
func tokenShapes() *regexp.Regexp {
	var named []string
	for _, shape := range strings.Split(catalog.LeakShapes, "|") {
		if !strings.HasPrefix(shape, "[") {
			named = append(named, shape)
		}
	}
	return regexp.MustCompile(strings.Join(named, "|"))
}

// invariant: rule fixtures-carry-no-secret (13.3 rule 16, SC-10): no JSON
// fixture carries a secret of a world, nor anything shaped like an issued
// token, so a page, a stream, a golden file or an agent's journal that
// kept one, pasted by the owner into a prompt or printed by a tool, would
// be caught here, not in a review.
func ruleFixturesCarryNoSecret(fixtures map[string]string, secrets []string) []finding {
	var out []finding
	shapes := tokenShapes()
	for _, rel := range slices.Sorted(maps.Keys(fixtures)) {
		for _, s := range secrets {
			if strings.Contains(fixtures[rel], s) {
				out = append(out, finding{"fixtures-carry-no-secret", rel, "carries a secret of the worlds"})
			}
		}
		if shapes.MatchString(fixtures[rel]) {
			out = append(out, finding{"fixtures-carry-no-secret", rel, "carries a token"})
		}
	}
	return out
}

func TestFixturesCarryNoSecret(t *testing.T) {
	root := repoRoot(t)
	fixtures, secrets := jsonFixtures(t, root), worldSecrets(t, root)
	if len(secrets) == 0 {
		t.Fatal("no world holds a secret: the rule proves nothing")
	}
	if len(fixtures) == 0 {
		t.Fatal("no JSON fixture found: the rule proves nothing")
	}
	for _, f := range ruleFixturesCarryNoSecret(fixtures, secrets) {
		t.Errorf("%s: %s", f.where, f.what)
	}
	t.Run("bites", func(t *testing.T) {
		mutant := map[string]string{"apps/engine/ui/testdata/pages/accounts.list.json": `{"token":"` + secrets[0] + `"}`}
		if len(ruleFixturesCarryNoSecret(mutant, secrets)) == 0 {
			t.Error("a page fixture that prints a world's secret passed the rule")
		}
		journal := map[string]string{"specs/usecases/agents/__fixtures__/journal.json": `{"task":"deploy with sk-ant-api03-` + strings.Repeat("k", 40) + `"}`}
		if len(ruleFixturesCarryNoSecret(journal, nil)) == 0 {
			t.Error("a journal fixture that kept a token passed the rule")
		}
	})
}
