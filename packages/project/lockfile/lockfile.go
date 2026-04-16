// Package lockfile owns spwn.lock: the committed, deterministic
// pin of every @spwn/* and github.com/* dependency the project uses.
//
// Format is Go-style, line-oriented text — one entry per line:
//
//	# spwn.lock — DO NOT EDIT
//	@spwn/unix v24.04 builtin
//	@spwn/git v2.43 builtin
//	github.com/jterrazz/research-skills v0.3.0 sha256:b7e12...
//
// Local (bare-name) refs never land in the lockfile — they are
// authored in-place under spwn/skills/, spwn/tools/, etc.
//
// Load/Save round-trip is deterministic: entries are sorted lexically
// so git diffs stay clean.
package lockfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileName is the canonical lockfile basename at the project root.
const FileName = "spwn.lock"

// Source identifies how an entry is resolved at build time.
type Source string

const (
	// SourceBuiltin means the pack is compiled into the spwn binary.
	SourceBuiltin Source = "builtin"
	// SourceGitHub means the pack comes from a GitHub repo.
	SourceGitHub Source = "github"
)

// Entry pins one dependency to a source and version.
type Entry struct {
	Version string
	Source  Source
}

// Lockfile is the parsed content of spwn.lock.
type Lockfile struct {
	Deps map[string]Entry
}

// Empty returns a fresh lockfile.
func Empty() *Lockfile {
	return &Lockfile{
		Deps: map[string]Entry{},
	}
}

// Path returns the canonical lockfile location for a project root.
func Path(projectRoot string) string {
	return filepath.Join(projectRoot, FileName)
}

// Exists reports whether a lockfile is present at the given project root.
func Exists(projectRoot string) bool {
	_, err := os.Stat(Path(projectRoot))
	return err == nil
}

// Load reads and parses the lockfile. Returns (nil, nil) when the file
// does not exist so callers can distinguish "never installed" from
// "corrupted".
func Load(projectRoot string) (*Lockfile, error) {
	data, err := os.ReadFile(Path(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	l := &Lockfile{Deps: map[string]Entry{}}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Legacy YAML format detection: if the first non-comment line
		// starts with "version:" or "deps:", fall back to YAML parse.
		if strings.HasPrefix(line, "version:") || strings.HasPrefix(line, "deps:") {
			return loadLegacyYAML(data)
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
		l.Deps[ref] = Entry{Version: version, Source: source}
	}
	return l, nil
}

// loadLegacyYAML handles the old YAML lockfile format for migration.
func loadLegacyYAML(data []byte) (*Lockfile, error) {
	// Minimal YAML parse — just extract the deps map.
	l := &Lockfile{Deps: map[string]Entry{}}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inDeps := false
	var currentRef string
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "deps:" {
			inDeps = true
			continue
		}
		if !inDeps {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 && trimmed != "" {
			break
		}
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			currentRef = strings.Trim(strings.TrimSuffix(trimmed, ":"), "\"")
			l.Deps[currentRef] = Entry{}
			continue
		}
		if indent == 4 && currentRef != "" {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			e := l.Deps[currentRef]
			switch key {
			case "version":
				e.Version = val
			case "source":
				e.Source = Source(val)
			}
			l.Deps[currentRef] = e
		}
	}
	return l, nil
}

// LoadOrEmpty is a convenience for callers that don't care whether the
// file existed.
func LoadOrEmpty(projectRoot string) (*Lockfile, error) {
	l, err := Load(projectRoot)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return Empty(), nil
	}
	return l, nil
}

// Save writes the lockfile deterministically: entries sorted lexically,
// one line per dep.
func Save(projectRoot string, l *Lockfile) error {
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
	if err := os.WriteFile(Path(projectRoot), []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}

// Add upserts an entry.
func (l *Lockfile) Add(ref string, entry Entry) {
	if l.Deps == nil {
		l.Deps = map[string]Entry{}
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

// Refs returns the sorted list of pinned refs.
func (l *Lockfile) Refs() []string {
	out := make([]string, 0, len(l.Deps))
	for k := range l.Deps {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
