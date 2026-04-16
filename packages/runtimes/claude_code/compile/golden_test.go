package claudecode

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"spwn.sh/packages/compile"
	"spwn.sh/packages/compile/source"
)

// TestGoldenFixtures runs the claude-code renderer against every
// sub-directory under testdata/ and compares the emitted Tree to the
// committed expected/ dir. This is the coverage moat for the
// renderer: regression on any byte fails in single-digit milliseconds.
//
// Layout of each fixture:
//
//	testdata/<name>/
//	  input/            -- a tiny spwn project (spwn.yaml + spwn/**)
//	  expected/         -- expected rendered tree (success case)
//	  expected-error.txt -- expected error substring (error case)
//
// Regenerate expected/ dirs with:
//
//	UPDATE_GOLDEN=1 go test -run TestGoldenFixtures ./packages/compile/runtimes/claudecode/...
func TestGoldenFixtures(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	update := os.Getenv("UPDATE_GOLDEN") == "1"

	runtime := &Runtime{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			fixtureDir := filepath.Join("testdata", name)
			inputDir := filepath.Join(fixtureDir, "input")
			expectedDir := filepath.Join(fixtureDir, "expected")
			expectedErrFile := filepath.Join(fixtureDir, "expected-error.txt")

			isErrorFixture := fileExists(expectedErrFile)

			src, loadErr := source.Load(inputDir)

			var renderErr error
			var tree *compile.Tree
			if loadErr == nil {
				input, inErr := source.ToCompileInput(src, "")
				if inErr != nil {
					renderErr = inErr
				} else {
					tree, renderErr = runtime.Render(input)
				}
			}

			gotErr := loadErr
			if gotErr == nil {
				gotErr = renderErr
			}

			if isErrorFixture {
				if gotErr == nil {
					t.Fatalf("expected an error for fixture %q, got none", name)
				}
				wantBytes, err := os.ReadFile(expectedErrFile)
				if err != nil {
					t.Fatalf("read %s: %v", expectedErrFile, err)
				}
				want := strings.TrimSpace(string(wantBytes))
				if update {
					if err := os.WriteFile(expectedErrFile, []byte(gotErr.Error()+"\n"), 0o644); err != nil {
						t.Fatalf("update %s: %v", expectedErrFile, err)
					}
					return
				}
				if !strings.Contains(gotErr.Error(), want) {
					t.Fatalf("error mismatch for %q:\n  want substring: %s\n  got:            %s",
						name, want, gotErr.Error())
				}
				return
			}

			// Success-path fixture
			if gotErr != nil {
				t.Fatalf("unexpected error for fixture %q: %v", name, gotErr)
			}

			if update {
				if err := os.RemoveAll(expectedDir); err != nil {
					t.Fatalf("rm %s: %v", expectedDir, err)
				}
				if err := tree.WriteTo(expectedDir); err != nil {
					t.Fatalf("write %s: %v", expectedDir, err)
				}
				return
			}

			assertTreeMatchesDir(t, tree, expectedDir)
		})
	}
}

// assertTreeMatchesDir walks expected/ and checks that every file is
// present in the tree with byte-identical content, and that the tree
// contains no extra files.
func assertTreeMatchesDir(t *testing.T, tree *compile.Tree, expectedDir string) {
	t.Helper()

	wantFiles := map[string][]byte{}
	err := filepath.Walk(expectedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(expectedDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		wantFiles[rel] = b
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", expectedDir, err)
	}

	gotPaths := tree.Paths()
	gotSet := map[string]struct{}{}
	for _, p := range gotPaths {
		gotSet[p] = struct{}{}
	}

	// Missing / mismatched files
	wantPaths := make([]string, 0, len(wantFiles))
	for p := range wantFiles {
		wantPaths = append(wantPaths, p)
	}
	sort.Strings(wantPaths)

	for _, p := range wantPaths {
		got, ok := tree.Get(p)
		if !ok {
			t.Errorf("tree missing %s", p)
			continue
		}
		want := wantFiles[p]
		if !bytes.Equal(got, want) {
			t.Errorf("content mismatch for %s:\n--- want ---\n%s\n--- got ---\n%s\n",
				p, truncate(string(want), 400), truncate(string(got), 400))
		}
	}

	// Extras
	for _, p := range gotPaths {
		if _, ok := wantFiles[p]; !ok {
			t.Errorf("tree has extra %s (not in expected/)", p)
		}
	}

	if t.Failed() {
		t.Log(fmt.Sprintf("tip: re-run with UPDATE_GOLDEN=1 to regenerate %s", expectedDir))
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
