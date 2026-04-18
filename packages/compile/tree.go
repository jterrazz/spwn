// Package compile is spwn's transpiler (historical package name;
// semantically: the source → runtime-specific tree step of the
// pipeline, i.e. what a compiler's front-end does). It translates a
// provider-neutral spwn project (spwn.yaml + spwn/agents/* + skills +
// hooks) into a runtime-specific file layout (Tree) that can then be
// materialised to disk or handed to packages/image for the compile
// step — linking the tree with tools into a Docker image.
//
// Think tsc: source = .ts files, intermediate = typed AST, emit =
// .js files under outDir. Spwn: source = the project tree, intermediate
// = the in-memory Tree, emit = a directory of files a concrete runtime
// understands.
//
// Pipeline:
//   packages/compile   — transpile source → Tree
//   packages/image     — compile Tree + tools → Docker image
//   packages/architect — spawn world from image
package compile

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Tree is a pure in-memory file layout produced by a Runtime. Paths
// are relative (no leading slash) and use forward slashes regardless
// of host OS. A Tree is the output of rendering a project for one
// target runtime.
//
// Tree is intentionally simple: a flat map of path to byte content.
// That fits today's needs (a few dozen files per world) and leaves
// room to grow into something streaming later if renderers ever need
// to emit gigabytes.
type Tree struct {
	files map[string][]byte
}

// New returns an empty Tree.
func New() *Tree {
	return &Tree{files: map[string][]byte{}}
}

// Add stores content at path. Paths are normalised to forward slashes
// and any leading "./" or "/" is stripped. Overwrites an existing
// entry at the same path.
func (t *Tree) Add(path string, content []byte) {
	t.files[normalisePath(path)] = content
}

// AddString is Add for string content. The stored bytes are a copy of
// the string's backing memory.
func (t *Tree) AddString(path string, content string) {
	t.Add(path, []byte(content))
}

// Has reports whether the Tree contains an entry at path.
func (t *Tree) Has(path string) bool {
	_, ok := t.files[normalisePath(path)]
	return ok
}

// Get returns the content stored at path and whether the entry exists.
func (t *Tree) Get(path string) ([]byte, bool) {
	b, ok := t.files[normalisePath(path)]
	return b, ok
}

// Paths returns every path in the Tree, sorted lexicographically.
func (t *Tree) Paths() []string {
	out := make([]string, 0, len(t.files))
	for p := range t.files {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Walk invokes fn for every entry in the Tree in sorted path order.
func (t *Tree) Walk(fn func(path string, content []byte)) {
	for _, p := range t.Paths() {
		fn(p, t.files[p])
	}
}

// WriteTo materialises the Tree under dir, creating parent directories
// as needed. Files are written with mode 0o644 and directories with
// 0o755. dir itself is created if missing.
func (t *Tree) WriteTo(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	for _, p := range t.Paths() {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, t.files[p], 0o644); err != nil {
			return fmt.Errorf("write %s: %w", full, err)
		}
	}
	return nil
}

// Tar writes the tree as an uncompressed tar stream. Output is
// deterministic: for the same input the same tar bytes are produced.
// Mode, modtime, and uid/gid are fixed; entries are written in the
// sorted path order exposed by Paths(). Paths are written unchanged
// (relative, no leading slash).
//
// Useful for feeding a compiled tree directly into a Docker build
// context without materialising to disk first.
func (t *Tree) Tar(w io.Writer) error {
	tw := tar.NewWriter(w)
	for _, path := range t.Paths() {
		content := t.files[path]
		hdr := &tar.Header{
			Name:    path,
			Mode:    0o644,
			Size:    int64(len(content)),
			ModTime: time.Unix(0, 0).UTC(),
			Uid:     0,
			Gid:     0,
			Format:  tar.FormatUSTAR,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("tar header %s: %w", path, err)
		}
		if _, err := tw.Write(content); err != nil {
			return fmt.Errorf("tar write %s: %w", path, err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar close: %w", err)
	}
	return nil
}

func normalisePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	return p
}
