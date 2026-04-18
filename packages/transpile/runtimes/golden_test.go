// Package runtimes_test holds the cross-runtime golden suite.
//
// Each fixture under testdata/<case>/ declares one `input/` source
// tree (a tiny spwn project) plus one expected-output directory per
// runtime the case exercises:
//
//	testdata/<name>/
//	  input/                        -- spwn.yaml + spwn/**
//	  output_<runtime>/             -- expected rendered Tree
//	  output_<runtime>_error.txt    -- expected error substring
//	                                   (used in place of output_<runtime>/
//	                                   when the renderer should fail)
//
// The test walks each case, discovers every runtime-output directory
// by scanning for the `output_*` naming convention, and drives the
// case through the matching runtime via transpile.Compile. Adding
// a new runtime means:
//
//  1. Blank-import its package below so its init() registers the
//     renderer.
//  2. Regenerate goldens with
//     UPDATE_GOLDEN=1 go test ./packages/transpile/runtimes/...
//
// Regenerating one runtime only: set UPDATE_GOLDEN=<runtime-name>
// (e.g. UPDATE_GOLDEN=claude-code).
package runtimes_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"spwn.sh/packages/transpile"
	"spwn.sh/packages/transpile/source"

	// Blank-import every runtime so its init() registers the renderer
	// With transpile.Compile. Order doesn't matter; each registration
	// Is keyed by Runtime.Name().
	_ "spwn.sh/packages/transpile/runtimes/claude_code"
)

// outputDirPrefix marks the directories the test treats as
// per-runtime expected-output trees. Anything under a case that
// Starts with this prefix names the target runtime after the
// Underscore: `output_claude_code/` → `claude-code`.
const outputDirPrefix = "output_"

// errorSuffix marks per-runtime expected-error fixtures for cases
// Where the renderer should fail instead of producing a tree. The
// File lives at testdata/<case>/output_<runtime>_error.txt.
const errorSuffix = "_error.txt"

// TestGoldenFixtures walks every case under testdata/, finds every
// Declared output_<runtime>/ (or output_<runtime>_error.txt), and
// Drives the same input through each matching runtime.
func TestGoldenFixtures(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	update := os.Getenv("UPDATE_GOLDEN")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		caseName := entry.Name()
		caseDir := filepath.Join("testdata", caseName)
		inputDir := filepath.Join(caseDir, "input")

		targets, err := discoverRuntimeTargets(caseDir)
		if err != nil {
			t.Fatalf("%s: discover runtimes: %v", caseName, err)
		}
		if len(targets) == 0 {
			t.Errorf("%s: no output_<runtime>/ or output_<runtime>_error.txt found", caseName)
			continue
		}

		for _, tg := range targets {
			subName := fmt.Sprintf("%s/%s", caseName, tg.runtime)
			t.Run(subName, func(t *testing.T) {
				src, loadErr := source.Load(inputDir)

				var renderErr error
				var tree *transpile.Tree
				if loadErr == nil {
					input, inErr := source.ToCompileInput(src, "")
					if inErr != nil {
						renderErr = inErr
					} else {
						tree, renderErr = transpile.Compile(tg.runtime, input)
					}
				}

				gotErr := loadErr
				if gotErr == nil {
					gotErr = renderErr
				}

				if tg.errorFile != "" {
					if gotErr == nil {
						t.Fatalf("expected an error, got none")
					}
					wantBytes, err := os.ReadFile(tg.errorFile)
					if err != nil {
						t.Fatalf("read %s: %v", tg.errorFile, err)
					}
					want := strings.TrimSpace(string(wantBytes))
					if shouldUpdate(update, tg.runtime) {
						if err := os.WriteFile(tg.errorFile, []byte(gotErr.Error()+"\n"), 0o644); err != nil {
							t.Fatalf("update %s: %v", tg.errorFile, err)
						}
						return
					}
					if !strings.Contains(gotErr.Error(), want) {
						t.Fatalf("error mismatch:\n  want substring: %s\n  got:            %s",
							want, gotErr.Error())
					}
					return
				}

				// Success-path: expected-tree directory must match the
				// Emitted Tree byte-for-byte.
				if gotErr != nil {
					t.Fatalf("unexpected error: %v", gotErr)
				}

				if shouldUpdate(update, tg.runtime) {
					if err := os.RemoveAll(tg.outputDir); err != nil {
						t.Fatalf("rm %s: %v", tg.outputDir, err)
					}
					if err := tree.WriteTo(tg.outputDir); err != nil {
						t.Fatalf("write %s: %v", tg.outputDir, err)
					}
					return
				}

				assertTreeMatchesDir(t, tree, tg.outputDir)
			})
		}
	}
}

// runtimeTarget pairs a discovered runtime name with the fixture
// Path the test should assert against — either a directory of
// Expected bytes or a single error substring file.
type runtimeTarget struct {
	runtime   string
	outputDir string // set when the target is a success-path tree
	errorFile string // set when the target is an expected-error file
}

// discoverRuntimeTargets walks a case directory and collects every
// Expected-output artifact. The outputDirPrefix naming convention
// Turns a filesystem entry into a runtime name by stripping the
// Prefix and converting `_` back to `-` (Go directory names can't
// Contain `-`, so claude-code lives at output_claude_code/).
func discoverRuntimeTargets(caseDir string) ([]runtimeTarget, error) {
	entries, err := os.ReadDir(caseDir)
	if err != nil {
		return nil, err
	}
	var targets []runtimeTarget
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, outputDirPrefix) {
			continue
		}
		rest := strings.TrimPrefix(name, outputDirPrefix)

		if strings.HasSuffix(rest, errorSuffix) && !e.IsDir() {
			runtimeID := strings.TrimSuffix(rest, errorSuffix)
			targets = append(targets, runtimeTarget{
				runtime:   pathNameToRuntime(runtimeID),
				errorFile: filepath.Join(caseDir, name),
			})
			continue
		}
		if e.IsDir() {
			targets = append(targets, runtimeTarget{
				runtime:   pathNameToRuntime(rest),
				outputDir: filepath.Join(caseDir, name),
			})
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].runtime < targets[j].runtime
	})
	return targets, nil
}

// pathNameToRuntime converts a filesystem-safe runtime identifier
// (`claude_code`) back to the canonical runtime.Name() form
// (`claude-code`) used by transpile.Compile.
func pathNameToRuntime(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}

// shouldUpdate checks whether UPDATE_GOLDEN selects this runtime.
// Unset: no update. "1" or "all": update everything. Any other
// value: update only the runtime whose name matches.
func shouldUpdate(env, runtime string) bool {
	switch env {
	case "":
		return false
	case "1", "all":
		return true
	default:
		return env == runtime
	}
}

// assertTreeMatchesDir walks outputDir and checks that every file
// Is present in the tree with byte-identical content, and that the
// Tree contains no extra files.
func assertTreeMatchesDir(t *testing.T, tree *transpile.Tree, outputDir string) {
	t.Helper()

	wantFiles := map[string][]byte{}
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(outputDir, path)
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
		t.Fatalf("walk %s: %v", outputDir, err)
	}

	gotPaths := tree.Paths()
	gotSet := map[string]struct{}{}
	for _, p := range gotPaths {
		gotSet[p] = struct{}{}
	}

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

	for _, p := range gotPaths {
		if _, ok := wantFiles[p]; !ok {
			t.Errorf("tree has extra %s (not in %s)", p, outputDir)
		}
	}

	if t.Failed() {
		t.Log(fmt.Sprintf("tip: re-run with UPDATE_GOLDEN=<runtime> (or UPDATE_GOLDEN=all) to regenerate %s", outputDir))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
