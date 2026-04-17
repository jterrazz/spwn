// Package dependency owns spwn.lock: the committed, deterministic
// pin of every spwn:* and github:*/* dependency the project uses.
//
// Format is Go-style, line-oriented text — one entry per line:
//
//	# spwn.lock — DO NOT EDIT
//	spwn:unix v24.04 builtin
//	spwn:git v2.43 builtin
//	github:jterrazz/research-skills v0.3.0 sha256:b7e12...
//
// Local (bare-name) refs never land in the lockfile — they are
// authored in-place under spwn/skills/, spwn/tools/, etc.
//
// Load/Save round-trip is deterministic: entries are sorted lexically
// so git diffs stay clean.
package dependency

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LockFileName is the canonical lockfile basename at the project root.
const LockFileName = "spwn.lock"

// Source identifies how an entry is resolved at build time.
type Source string

const (
	// SourceBuiltin means the dependency is compiled into the spwn binary.
	SourceBuiltin Source = "builtin"
	// SourceGitHub means the dependency comes from a GitHub repo.
	SourceGitHub Source = "github"
)

// LockEntry pins one dependency to a source and version.
type LockEntry struct {
	Version string
	Source  Source
}

// Lockfile is the parsed content of spwn.lock.
type Lockfile struct {
	Deps map[string]LockEntry
}

// Empty returns a fresh dependency.
func EmptyLockfile() *Lockfile {
	return &Lockfile{
		Deps: map[string]LockEntry{},
	}
}

// Path returns the canonical lockfile location for a project root.
func LockfilePath(projectRoot string) string {
	return filepath.Join(projectRoot, LockFileName)
}

// Exists reports whether a lockfile is present at the given project root.
func LockfileExists(projectRoot string) bool {
	_, err := os.Stat(LockfilePath(projectRoot))
	return err == nil
}

// Load reads and parses the dependency. Returns (nil, nil) when the file
// does not exist so callers can distinguish "never installed" from
// "corrupted".
func LoadLockfile(projectRoot string) (*Lockfile, error) {
	data, err := os.ReadFile(LockfilePath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", LockFileName, err)
	}
	l := &Lockfile{Deps: map[string]LockEntry{}}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[0]
		version := parts[1]
		source := SourceBuiltin
		if len(parts) >= 3 {
			source = Source(parts[2])
		}
		l.Deps[ref] = LockEntry{Version: version, Source: source}
	}
	return l, nil
}

// LoadOrEmpty is a convenience for callers that don't care whether the
// file existed.
func LoadLockfileOrEmpty(projectRoot string) (*Lockfile, error) {
	l, err := LoadLockfile(projectRoot)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return EmptyLockfile(), nil
	}
	return l, nil
}

// Save writes the lockfile deterministically: entries sorted lexically,
// one line per dep.
func SaveLockfile(projectRoot string, l *Lockfile) error {
	if l == nil {
		return fmt.Errorf("nil lockfile")
	}
	var b strings.Builder
	b.WriteString("# spwn.lock — DO NOT EDIT\n")
	keys := make([]string, 0, len(l.Deps))
	for k := range l.Deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		e := l.Deps[k]
		version := e.Version
		if version == "" {
			version = "latest"
		}
		fmt.Fprintf(&b, "%s %s %s\n", k, version, e.Source)
	}
	if err := os.WriteFile(LockfilePath(projectRoot), []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", LockFileName, err)
	}
	return nil
}

// Add upserts an entry.
func (l *Lockfile) Add(ref string, entry LockEntry) {
	if l.Deps == nil {
		l.Deps = map[string]LockEntry{}
	}
	l.Deps[ref] = entry
}

// Remove deletes an entry. No-op when absent.
func (l *Lockfile) Remove(ref string) {
	delete(l.Deps, ref)
}

// Has reports whether the ref is pinned.
func (l *Lockfile) Has(ref string) bool {
	_, ok := l.Deps[ref]
	return ok
}

// Refs returns the sorted list of pinned dependency.
func (l *Lockfile) Refs() []string {
	out := make([]string, 0, len(l.Deps))
	for k := range l.Deps {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
