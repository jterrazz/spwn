// Package resolve walks markdown @-imports to produce the full file
// set a single agent brings into a world at runtime.
//
// Claude Code treats `@path/to/file.md` inside a markdown file as an
// include directive - the runtime effectively inlines the referenced
// file's contents. We follow the same convention. Given a starting
// file (typically CLAUDE.md), Walk returns every file the runtime
// would read and any broken references it hit along the way.
package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// importRe matches `@path` references at word boundaries. It's
// intentionally conservative: `@foo` is an import, `email@foo.com`
// is not (the `@` follows an alphanumeric).
var importRe = regexp.MustCompile(`(^|[^\w/])@([\w.\-/]+\.md)\b`)

// Result is the outcome of a single Walk - every file visited, every
// broken reference found, and the order in which files were first
// discovered (useful for build artifacts and error reporting).
type Result struct {
	// Root is the directory used to resolve relative references.
	Root string

	// Visited lists every file Walk successfully read, in discovery
	// order. Always includes the starting file first.
	Visited []string

	// Missing lists broken @-imports encountered. Each entry is a
	// Reference where the target file could not be opened.
	Missing []Reference

	// Cycles lists self-referential or circular import chains found
	// during the walk. An empty slice means the import graph is a
	// DAG.
	Cycles [][]string
}

// Reference is one @-import found in a source file.
type Reference struct {
	// Source is the file that contained the reference (absolute
	// path).
	Source string

	// Target is the @path as written in the source (kept verbatim
	// for error messages).
	Target string

	// ResolvedPath is Target joined with Root (or Source's dir).
	// Set even when the target file is missing, so callers have a
	// concrete path to report.
	ResolvedPath string
}

// Walk follows @-imports starting at startFile and returns every file
// reachable through the import graph. Errors opening the starting
// file are returned; broken @-imports inside valid files surface as
// entries in Result.Missing. Cycles are detected and reported in
// Result.Cycles - they do NOT block the walk.
//
// The root parameter is used to resolve references that begin with
// the agent root (e.g. `@core/profile.md`). References without a
// leading `/` are treated as relative to the containing file's
// directory first, then as relative to root.
func Walk(root, startFile string) (*Result, error) {
	absStart, err := filepath.Abs(startFile)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", startFile, err)
	}
	if _, err := os.Stat(absStart); err != nil {
		return nil, fmt.Errorf("stat %s: %w", absStart, err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root %s: %w", root, err)
	}

	r := &Result{Root: absRoot}
	visiting := map[string]int{} // path → depth on the current path
	visited := map[string]bool{}

	var walk func(path string, stack []string)
	walk = func(path string, stack []string) {
		// Cycle check first: if the target is already on the
		// current call stack, it's a back-edge.
		if _, onStack := visiting[path]; onStack {
			cycleStart := 0
			for i, p := range stack {
				if p == path {
					cycleStart = i
					break
				}
			}
			cycle := append([]string{}, stack[cycleStart:]...)
			cycle = append(cycle, path)
			r.Cycles = append(r.Cycles, cycle)
			return
		}
		// Already fully processed on a previous branch: skip.
		if visited[path] {
			return
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		visiting[path] = len(stack)
		stack = append(stack, path)

		r.Visited = append(r.Visited, path)
		visited[path] = true

		for _, ref := range extractImports(path, string(data)) {
			candidates := resolveCandidates(absRoot, path, ref.Target)
			resolved := ""
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					resolved = c
					break
				}
			}
			if resolved == "" {
				ref.ResolvedPath = candidates[0]
				r.Missing = append(r.Missing, ref)
				continue
			}
			ref.ResolvedPath = resolved
			walk(resolved, stack)
		}

		delete(visiting, path)
	}

	walk(absStart, nil)
	return r, nil
}

// extractImports pulls every @-reference out of the given markdown
// body. Source is the path to the containing file, preserved in the
// returned References.
func extractImports(source, body string) []Reference {
	var out []Reference
	seen := map[string]bool{}
	for _, m := range importRe.FindAllStringSubmatch(body, -1) {
		if len(m) < 3 {
			continue
		}
		target := strings.TrimSpace(m[2])
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		out = append(out, Reference{Source: source, Target: target})
	}
	return out
}

// resolveCandidates returns the list of absolute paths Walk should
// try, in order of preference, for a given @-import target.
func resolveCandidates(root, source, target string) []string {
	if filepath.IsAbs(target) {
		return []string{target}
	}
	sourceDir := filepath.Dir(source)
	return []string{
		filepath.Join(sourceDir, target),
		filepath.Join(root, target),
	}
}
