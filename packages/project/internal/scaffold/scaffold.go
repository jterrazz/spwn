// Package scaffold materializes a fresh spwn project on disk.
//
// The templates live next to this file as *.tmpl and are embedded at
// build time, so a new project needs no network access and no external
// template files to work.
package scaffold

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/dependency"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/project/internal/layout"
	intmanifest "spwn.sh/packages/project/internal/manifest"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Opts configures Init. See manifest.InitOpts for field docs.
type Opts struct {
	Name        string
	Force       bool
	NoGitignore bool
	// Backend, when non-empty, is emitted as `runtime.backend:` in the
	// scaffolded agent.yaml. Empty leaves the scaffold runtime-agnostic
	// so resolution happens at spawn time based on project-wide config
	// and the user's authenticated backends.
	Backend string
}

// Init materializes a fresh spwn project under dir. The directory
// must already exist.
func Init(dir string, opts Opts) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", dir, err)
	}
	project := platform.Within(absDir)
	defer project.Close()
	if !project.IsDir(".") {
		return fmt.Errorf("%s is not a directory", absDir)
	}

	manifestPath := filepath.Join(absDir, "spwn.yaml")
	if _, statErr := project.Stat("spwn.yaml"); statErr == nil && !opts.Force {
		return fmt.Errorf("spwn.yaml already exists at %s.\nUse --force to overwrite", manifestPath)
	}

	name := opts.Name
	if name == "" {
		name = defaultName(absDir)
	}
	files, err := Files(name, opts.Backend)
	if err != nil {
		return err
	}
	rels := make([]string, 0, len(files))
	for rel := range files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		dst := filepath.FromSlash(rel)
		if err := project.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", rel, err)
		}
		if err := project.WriteFile(dst, files[rel], 0o644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}

	if !opts.NoGitignore {
		if err := appendGitignore(project); err != nil {
			return fmt.Errorf("update .gitignore: %w", err)
		}
	}

	return nil
}

// AddWorldOpts configures AddWorld. All fields are optional except
// the world name passed as a separate argument.
type AddWorldOpts struct {
	// Agents is the list of agent names this world deploys. Empty is
	// allowed at scaffold time but `station project plan` will flag it.
	Agents []string
	// Workspaces is the list of workspace mount specs. Empty defaults
	// to ["."] so the project root is mounted at /workspace.
	Workspaces []string
}

// AddWorld inserts a new entry under spwn.yaml#worlds with the given
// name and options. Idempotent: a no-op if an entry with that name
// already exists. Used by `spwn world create <name>`.
func AddWorld(manifestPath, name string, opts AddWorldOpts) error {
	data, err := readManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", manifestPath, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", manifestPath, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("unexpected yaml structure in %s", manifestPath)
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("spwn.yaml root must be a mapping")
	}

	worlds := findMapValue(root, "worlds")
	if worlds == nil {
		worlds = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "worlds"},
			worlds,
		)
	}
	if worlds.Kind != yaml.MappingNode {
		return fmt.Errorf("spwn.yaml#worlds must be a mapping")
	}
	if findMapValue(worlds, name) != nil {
		return nil
	}

	workspaces := opts.Workspaces
	if len(workspaces) == 0 {
		workspaces = []string{"."}
	}

	entry := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	entry.Content = append(entry.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "agents"},
		flowSeq(opts.Agents),
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "workspaces"},
		flowSeq(workspaces),
	)
	worlds.Content = append(worlds.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
		entry,
	)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("re-encode %s: %w", manifestPath, err)
	}
	return writeManifest(manifestPath, out)
}

// RemoveWorld drops the named entry from spwn.yaml#worlds. Returns
// an error wrapping ErrWorldNotFound if no such entry exists. Used
// by `spwn world rm <name>`.
func RemoveWorld(manifestPath, name string) error {
	data, err := readManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", manifestPath, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", manifestPath, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("unexpected yaml structure in %s", manifestPath)
	}
	root := doc.Content[0]
	worlds := findMapValue(root, "worlds")
	if worlds == nil || worlds.Kind != yaml.MappingNode {
		return ErrWorldNotFound
	}
	for i := 0; i+1 < len(worlds.Content); i += 2 {
		if worlds.Content[i].Value == name {
			worlds.Content = append(worlds.Content[:i], worlds.Content[i+2:]...)
			out, err := yaml.Marshal(&doc)
			if err != nil {
				return fmt.Errorf("re-encode %s: %w", manifestPath, err)
			}
			return writeManifest(manifestPath, out)
		}
	}
	return ErrWorldNotFound
}

// ErrWorldNotFound is returned by RemoveWorld when the named world
// does not exist in spwn.yaml.
var ErrWorldNotFound = fmt.Errorf("world not found in spwn.yaml")

func flowSeq(values []string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	for _, v := range values {
		n.Content = append(n.Content, &yaml.Node{
			Kind: yaml.ScalarNode, Tag: "!!str", Value: v,
		})
	}
	return n
}

func findMapValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// EncodeManifest is a small helper used by tests / CLI to render a
// Manifest struct back to YAML bytes.
func EncodeManifest(m *intmanifest.Manifest) ([]byte, error) {
	return yaml.Marshal(m)
}

type fileSpec struct {
	src string
	dst string
}

type templateData struct {
	Name    string
	Backend string
}

// Files returns the files of a fresh project named name, by slash path
// from its root: the templates rendered, a .gitkeep in each layer folder
// and the seeded spwn.lock. Its .gitignore is not among them. backend, when
// non-empty, is the starter agent's runtime.backend.
func Files(name, backend string) (map[string][]byte, error) {
	data := templateData{Name: name, Backend: backend}
	l := layout.Default()
	neo := path.Join(l.Folder(layout.Agents), "neo")
	templates := []fileSpec{
		{"templates/spwn.yaml.tmpl", "spwn.yaml"},
		{"templates/agent.yaml.tmpl", path.Join(neo, "agent.yaml")},
		{"templates/AGENTS.md.tmpl", path.Join(neo, "AGENTS.md")},
		{"templates/SOUL.md.tmpl", path.Join(neo, "SOUL.md")},
		{"templates/skill.focus.md.tmpl", path.Join(l.Folder(layout.Skills), "focus.md")},
		{"templates/tool.greet.yaml.tmpl", path.Join(l.Folder(layout.Tools), "greet", "tool.yaml")},
		{"templates/hook.session-banner.yaml.tmpl", path.Join(l.Folder(layout.Hooks), "session-banner.yaml")},
		{"templates/command.refactor.md.tmpl", path.Join(l.Folder(layout.Commands), "refactor.md")},
	}
	out := map[string][]byte{}
	for _, f := range templates {
		b, err := renderTemplate(f.src, data)
		if err != nil {
			return nil, err
		}
		out[f.dst] = b
	}
	for _, rel := range []string{path.Join(neo, "playbooks"), path.Join(neo, "journal"), "knowledge"} {
		out[rel+"/.gitkeep"] = []byte{}
	}
	lock := dependency.EmptyLockfile()
	for _, ref := range []string{"spwn:unix", "spwn:git", "spwn:python"} {
		lock.Add(ref, dependency.LockEntry{Source: dependency.SourceBuiltin})
	}
	out[dependency.LockFileName] = lock.Encode()
	return out, nil
}

func renderTemplate(srcRel string, data templateData) ([]byte, error) {
	raw, err := templatesFS.ReadFile(srcRel)
	if err != nil {
		return nil, fmt.Errorf("load template %s: %w", srcRel, err)
	}
	tmpl, err := template.New(srcRel).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", srcRel, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render template %s: %w", srcRel, err)
	}
	return buf.Bytes(), nil
}

func defaultName(absDir string) string {
	base := filepath.Base(absDir)
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == '_', r == ' ', r == '.':
			b.WriteByte('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "my-project"
	}
	return slug
}

// Gitignore returns the bytes of a .gitignore that held existing, with
// the `.spwn/` entry added when it is not there yet; existing itself
// when it is.
func Gitignore(existing []byte) []byte {
	needle := ".spwn/"
	if bytes.Contains(existing, []byte("\n"+needle)) || bytes.HasPrefix(existing, []byte(needle+"\n")) || bytes.Equal(existing, []byte(needle)) {
		return existing
	}
	var out bytes.Buffer
	out.Write(existing)
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		out.WriteByte('\n')
	}
	if len(existing) > 0 {
		out.WriteByte('\n')
		out.WriteString("# Spwn local state\n")
	}
	out.WriteString(needle)
	out.WriteByte('\n')
	return out.Bytes()
}

func appendGitignore(project *platform.Tree) error {
	existing, err := project.ReadFile(".gitignore")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	out := Gitignore(existing)
	if bytes.Equal(out, existing) {
		return nil
	}
	return project.WriteFile(".gitignore", out, 0o644)
}

func readManifest(manifestPath string) ([]byte, error) {
	project := platform.Within(filepath.Dir(manifestPath))
	defer project.Close()
	return project.ReadFile(filepath.Base(manifestPath))
}

func writeManifest(manifestPath string, data []byte) error {
	project := platform.Within(filepath.Dir(manifestPath))
	defer project.Close()
	return project.WriteFile(filepath.Base(manifestPath), data, 0o644)
}
