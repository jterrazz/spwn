package catalog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/compile"
	"spwn.sh/packages/compile/base"
	"spwn.sh/packages/runtimes"
)

// TestCatalogBundles walks every subdir under testdata/bundles and
// asserts what each catalog entry contributes to the final image:
//
//   - the resolved tool list (after transitive-dependency expansion)
//   - the skill files that land under /world/skills/ in the image
//   - the rendered Dockerfile (the bytes `docker build` would see),
//     so regressions in apt packages, RUN lines, or env exports
//     surface as a byte-level golden diff
//
// Layout of each fixture:
//
//	testdata/bundles/<name>/
//	  input.yaml          — tools: [...] list of refs to resolve
//	  expected/
//	    tools.txt         — resolved tool names, one per line, in order
//	    Dockerfile        — full rendered Dockerfile bytes
//	    skills/           — mirror of the /world/skills/ tree the image
//	                        will carry (paths relative here; content
//	                        byte-identical to what CollectSkills emits)
//
// Regenerate `expected/` with:
//
//	UPDATE_GOLDEN=1 go test -run TestCatalogBundles ./catalog/...
//
// This is the gap-coverage for the catalog→image path: without it,
// a silent regression in a catalog entry's bundled skills file (or
// the INDEX.md template) would slip through to production.
func TestCatalogBundles(t *testing.T) {
	entries, err := os.ReadDir("testdata/bundles")
	if err != nil {
		t.Fatalf("read testdata/bundles: %v", err)
	}

	update := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			fixtureDir := filepath.Join("testdata/bundles", name)

			// Load input spec.
			inputBytes, err := os.ReadFile(filepath.Join(fixtureDir, "input.yaml"))
			if err != nil {
				t.Fatalf("read input.yaml: %v", err)
			}
			var in struct {
				Tools []string `yaml:"tools"`
			}
			if err := yaml.Unmarshal(inputBytes, &in); err != nil {
				t.Fatalf("parse input.yaml: %v", err)
			}

			// Build a fresh registry from the embedded catalog and
			// resolve the requested refs. This exercises the same
			// code path `packages/architect` runs at spawn time.
			reg := compile.NewRegistry()
			for _, tool := range All {
				if err := reg.Register(tool); err != nil {
					t.Fatalf("register %s: %v", tool.Name(), err)
				}
			}
			// Runtime adapters (e.g. spwn:claude-code) are shipped in
			// packages/runtimes, not the catalog. The CLI wires both
			// Into the same registry at startup; the test mirrors that
			// So tools with transitive deps on runtimes resolve.
			for _, rt := range runtimes.All {
				if err := reg.Register(rt); err != nil {
					t.Fatalf("register %s: %v", rt.Name(), err)
				}
			}
			resolved, err := reg.Resolve(in.Tools)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}

			// Actual outputs captured from the same APIs the image
			// Builder uses at build time.
			got := struct {
				tools      string
				skills     map[string][]byte
				dockerfile []byte
			}{}
			{
				var b strings.Builder
				for _, t := range resolved {
					b.WriteString(t.Name())
					b.WriteString("\n")
				}
				got.tools = b.String()
			}
			skills, err := compile.CollectSkills(resolved)
			if err != nil {
				t.Fatalf("CollectSkills: %v", err)
			}
			got.skills = skills
			// Render the Dockerfile with a pinned image version so
			// The golden stays stable across version bumps of the
			// Base tree — the label line takes the constant below
			// And everything else derives from the resolved tools.
			got.dockerfile = compile.GenerateDockerfile(
				base.WorldDockerfile,
				compile.ToolsToInputs(resolved),
				"v-test",
			)

			expectedDir := filepath.Join(fixtureDir, "expected")

			if update {
				if err := os.RemoveAll(expectedDir); err != nil {
					t.Fatalf("rm %s: %v", expectedDir, err)
				}
				if err := os.MkdirAll(expectedDir, 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", expectedDir, err)
				}
				if err := os.WriteFile(filepath.Join(expectedDir, "tools.txt"), []byte(got.tools), 0o644); err != nil {
					t.Fatalf("write tools.txt: %v", err)
				}
				if err := os.WriteFile(filepath.Join(expectedDir, "Dockerfile"), got.dockerfile, 0o644); err != nil {
					t.Fatalf("write Dockerfile: %v", err)
				}
				for path, content := range got.skills {
					rel := strings.TrimPrefix(path, "/world/")
					dst := filepath.Join(expectedDir, rel)
					if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
						t.Fatalf("mkdir %s: %v", dst, err)
					}
					if err := os.WriteFile(dst, content, 0o644); err != nil {
						t.Fatalf("write %s: %v", dst, err)
					}
				}
				return
			}

			// Compare tool list.
			wantTools, err := os.ReadFile(filepath.Join(expectedDir, "tools.txt"))
			if err != nil {
				t.Fatalf("read tools.txt: %v", err)
			}
			if string(wantTools) != got.tools {
				t.Errorf("resolved tools mismatch:\n--- want ---\n%s--- got ---\n%s", wantTools, got.tools)
			}

			// Compare rendered Dockerfile.
			wantDockerfile, err := os.ReadFile(filepath.Join(expectedDir, "Dockerfile"))
			if err != nil {
				t.Fatalf("read Dockerfile: %v", err)
			}
			if !bytes.Equal(wantDockerfile, got.dockerfile) {
				t.Errorf("Dockerfile mismatch:\n--- want ---\n%s\n--- got ---\n%s",
					truncateBytes(wantDockerfile, 800), truncateBytes(got.dockerfile, 800))
			}

			// Compare skill tree. Walk the expected/skills dir
			// And check each file against CollectSkills output.
			wantSkills := map[string][]byte{}
			skillsRoot := filepath.Join(expectedDir, "skills")
			if _, err := os.Stat(skillsRoot); err == nil {
				err = filepath.Walk(skillsRoot, func(path string, info os.FileInfo, werr error) error {
					if werr != nil {
						return werr
					}
					if info.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(expectedDir, path)
					if err != nil {
						return err
					}
					b, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					// Map the on-disk relative path back to the
					// /world/ prefix CollectSkills emits.
					wantSkills["/world/"+filepath.ToSlash(rel)] = b
					return nil
				})
				if err != nil {
					t.Fatalf("walk expected skills: %v", err)
				}
			}

			// Ordered comparison for readable diffs.
			paths := make([]string, 0, len(wantSkills)+len(got.skills))
			seen := map[string]bool{}
			for k := range wantSkills {
				if !seen[k] {
					paths = append(paths, k)
					seen[k] = true
				}
			}
			for k := range got.skills {
				if !seen[k] {
					paths = append(paths, k)
					seen[k] = true
				}
			}
			sort.Strings(paths)

			for _, p := range paths {
				want, wantOK := wantSkills[p]
				actual, gotOK := got.skills[p]
				switch {
				case !wantOK:
					t.Errorf("unexpected skill file produced: %s\n%s", p, truncateBytes(actual, 200))
				case !gotOK:
					t.Errorf("missing skill file: %s\n(expected content:\n%s)", p, truncateBytes(want, 200))
				case !bytes.Equal(want, actual):
					t.Errorf("content mismatch for %s:\n--- want ---\n%s\n--- got ---\n%s",
						p, truncateBytes(want, 400), truncateBytes(actual, 400))
				}
			}

			if t.Failed() {
				t.Log(fmt.Sprintf("tip: re-run with UPDATE_GOLDEN=1 to regenerate %s", expectedDir))
			}
		})
	}
}

func truncateBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "...(truncated)"
}
