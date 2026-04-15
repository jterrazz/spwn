// Package lockfile owns spwn.lock.yaml: the committed, deterministic
// pin of every @spwn/* and @<org>/* reference the project depends on.
//
// The lockfile mirrors the dependency lists in each agent.yaml but
// collapses them into a single project-wide record. Local (bare-name)
// refs never land in the lockfile — they are authored in-place under
// spwn/tools/ and spwn/skills/ and have no version to pin.
//
// Shape:
//
//	version: 1
//	tools:
//	  "@spwn/unix":
//	    version: "24.04"
//	    source: builtin
//	  "@spwn/git":
//	    version: "2.43"
//	    source: builtin
//	plugins:
//	  "@spwn/mempalace":
//	    version: "0.1.0"
//	    source: builtin
//	skills: {}
//
// `source: builtin` means the pack is compiled into the spwn binary
// and needs no on-disk cache. `source: registry` is reserved for the
// future community registry — resolved to <root>/.spwn/packs/@<org>/<name>/.
// Nothing emits registry entries today.
//
// Load/Save round-trip is deterministic: keys are sorted lexically so
// git diffs stay clean.
package lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// CurrentVersion is the lockfile schema version. Bump when the shape
// changes in a way Load needs to migrate.
const CurrentVersion = 1

// FileName is the canonical lockfile basename at the project root.
const FileName = "spwn.lock.yaml"

// Source identifies how an entry is resolved at build time.
type Source string

const (
	// SourceBuiltin means the pack is compiled into the spwn binary.
	// No on-disk cache, no download.
	SourceBuiltin Source = "builtin"
	// SourceRegistry means the pack lives under .spwn/packs/@<org>/<name>/.
	// Reserved for the future community registry.
	SourceRegistry Source = "registry"
)

// Entry pins one dependency to a source and version.
type Entry struct {
	Version string `yaml:"version"`
	Source  Source `yaml:"source"`
}

// Lockfile is the parsed content of spwn.lock.yaml.
type Lockfile struct {
	Version int              `yaml:"version"`
	Tools   map[string]Entry `yaml:"tools"`
	Plugins map[string]Entry `yaml:"plugins"`
	Skills  map[string]Entry `yaml:"skills"`
}

// Empty returns a fresh lockfile at the current schema version.
func Empty() *Lockfile {
	return &Lockfile{
		Version: CurrentVersion,
		Tools:   map[string]Entry{},
		Plugins: map[string]Entry{},
		Skills:  map[string]Entry{},
	}
}

// Path returns the canonical lockfile location for a project root.
func Path(projectRoot string) string {
	return filepath.Join(projectRoot, FileName)
}

// Exists reports whether a lockfile is present at the given project
// root. Any stat error (including permission errors) is treated as
// "does not exist" — the caller decides whether that's fatal.
func Exists(projectRoot string) bool {
	_, err := os.Stat(Path(projectRoot))
	return err == nil
}

// Load reads and parses the lockfile at projectRoot. Returns (nil, nil)
// when the file does not exist so callers can distinguish "never
// installed" from "file corrupted".
func Load(projectRoot string) (*Lockfile, error) {
	data, err := os.ReadFile(Path(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	var l Lockfile
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parse %s: %w", FileName, err)
	}
	if l.Tools == nil {
		l.Tools = map[string]Entry{}
	}
	if l.Plugins == nil {
		l.Plugins = map[string]Entry{}
	}
	if l.Skills == nil {
		l.Skills = map[string]Entry{}
	}
	if l.Version == 0 {
		l.Version = CurrentVersion
	}
	if l.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported lockfile version %d (expected %d)", l.Version, CurrentVersion)
	}
	return &l, nil
}

// LoadOrEmpty is a convenience for callers that don't care whether
// the file existed. A missing file yields a fresh lockfile; a parse
// error still propagates.
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

// Save writes the lockfile deterministically: keys sorted lexically
// so git diffs only move when dependencies actually change.
func Save(projectRoot string, l *Lockfile) error {
	if l == nil {
		return fmt.Errorf("nil lockfile")
	}
	if l.Version == 0 {
		l.Version = CurrentVersion
	}
	// Build a plain map-of-maps with sorted keys to force deterministic
	// marshalling; yaml.v3 preserves insertion order for MapSlice but
	// not for `map[string]`, so go through a hand-built node tree.
	root := &yaml.Node{Kind: yaml.DocumentNode}
	body := &yaml.Node{Kind: yaml.MappingNode}
	root.Content = []*yaml.Node{body}

	addScalar := func(parent *yaml.Node, key string, value *yaml.Node) {
		parent.Content = append(parent.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			value,
		)
	}

	addScalar(body, "version", &yaml.Node{
		Kind: yaml.ScalarNode, Tag: "!!int",
		Value: fmt.Sprintf("%d", l.Version),
	})
	addScalar(body, "tools", mapToNode(l.Tools))
	addScalar(body, "plugins", mapToNode(l.Plugins))
	addScalar(body, "skills", mapToNode(l.Skills))

	data, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	if err := os.WriteFile(Path(projectRoot), data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}

func mapToNode(m map[string]Entry) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	if len(m) == 0 {
		// Emit `{}` instead of `null` so empty sections round-trip cleanly.
		node.Style = yaml.FlowStyle
		return node
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		e := m[k]
		entryNode := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "version"},
			{Kind: yaml.ScalarNode, Value: e.Version, Style: yaml.DoubleQuotedStyle},
			{Kind: yaml.ScalarNode, Value: "source"},
			{Kind: yaml.ScalarNode, Value: string(e.Source)},
		}}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k, Style: yaml.DoubleQuotedStyle},
			entryNode,
		)
	}
	return node
}

// Kind selects which dependency map an operation targets.
type Kind int

const (
	KindTool Kind = iota
	KindPlugin
	KindSkill
)

func (l *Lockfile) section(k Kind) map[string]Entry {
	switch k {
	case KindTool:
		return l.Tools
	case KindPlugin:
		return l.Plugins
	case KindSkill:
		return l.Skills
	}
	return nil
}

// Add upserts an entry. Passing an empty version is valid — callers
// that don't track versions yet record "" as the pin.
func (l *Lockfile) Add(k Kind, ref string, entry Entry) {
	m := l.section(k)
	if m == nil {
		return
	}
	m[ref] = entry
}

// Remove deletes an entry. No-op when the ref is absent.
func (l *Lockfile) Remove(k Kind, ref string) {
	m := l.section(k)
	if m == nil {
		return
	}
	delete(m, ref)
}

// Has reports whether the ref is pinned in the given section.
func (l *Lockfile) Has(k Kind, ref string) bool {
	m := l.section(k)
	if m == nil {
		return false
	}
	_, ok := m[ref]
	return ok
}

// RefsIn returns the sorted list of refs in the given section. Useful
// for deterministic iteration in error messages and tests.
func (l *Lockfile) RefsIn(k Kind) []string {
	m := l.section(k)
	if m == nil {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
