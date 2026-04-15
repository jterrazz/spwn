// Package scaffold materializes a fresh spwn project on disk.
//
// The templates live next to this file as *.tmpl and are embedded at
// build time, so spwn init needs no network access and no external
// template files to work.
package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	intmanifest "spwn.sh/packages/project/internal/manifest"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Opts configures Init. See manifest.InitOpts for field docs.
type Opts struct {
	Name        string
	Force       bool
	NoGitignore bool
}

// Init materializes a fresh spwn project under dir. The directory
// must already exist.
func Init(dir string, opts Opts) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", dir, err)
	}
	if info, statErr := os.Stat(absDir); statErr != nil || !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absDir)
	}

	manifestPath := filepath.Join(absDir, "spwn.yaml")
	if _, statErr := os.Stat(manifestPath); statErr == nil && !opts.Force {
		return fmt.Errorf("spwn.yaml already exists at %s.\nUse --force to overwrite", manifestPath)
	}

	name := opts.Name
	if name == "" {
		name = defaultName(absDir)
	}
	data := templateData{Name: name}

	files := []fileSpec{
		{"templates/spwn.yaml.tmpl", "spwn.yaml"},
		{"templates/agent.yaml.tmpl", "spwn/agents/neo/agent.yaml"},
		{"templates/AGENT.md.tmpl", "spwn/agents/neo/AGENT.md"},
		{"templates/profile.md.tmpl", "spwn/agents/neo/identity/profile.md"},
	}
	for _, f := range files {
		if err := writeTemplate(absDir, f.src, f.dst, data); err != nil {
			return err
		}
	}

	// Empty layer dirs: preserve them with .gitkeep so git tracks them.
	layerDirs := []string{
		"spwn/agents/neo/skills",
		"spwn/agents/neo/knowledge",
		"spwn/agents/neo/playbooks",
		"spwn/agents/neo/journal",
	}
	for _, rel := range layerDirs {
		dst := filepath.Join(absDir, rel, ".gitkeep")
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", rel, err)
		}
		if err := os.WriteFile(dst, nil, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}

	stateDir := filepath.Join(absDir, ".spwn")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("mkdir .spwn: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "state.json"), []byte("{}\n"), 0o644); err != nil {
		return fmt.Errorf("write .spwn/state.json: %w", err)
	}

	if !opts.NoGitignore {
		if err := appendGitignore(absDir); err != nil {
			return fmt.Errorf("update .gitignore: %w", err)
		}
	}

	return nil
}

// AddAgentWorld inserts a `worlds.<agent>: { agents: [<agent>], workspaces: [.] }`
// entry into spwn.yaml. Idempotent: a no-op if the entry already
// exists. Used by `spwn agent new <name>` (unless --no-world).
func AddAgentWorld(manifestPath, agentName string) error {
	data, err := os.ReadFile(manifestPath)
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
		// Create a worlds: map.
		worlds = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "worlds"},
			worlds,
		)
	}
	if worlds.Kind != yaml.MappingNode {
		return fmt.Errorf("spwn.yaml#worlds must be a mapping")
	}
	// Idempotency: already present?
	if findMapValue(worlds, agentName) != nil {
		return nil
	}

	entry := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	entry.Content = append(entry.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "agents"},
		flowSeq([]string{agentName}),
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "workspaces"},
		flowSeq([]string{"."}),
	)
	worlds.Content = append(worlds.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: agentName},
		entry,
	)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("re-encode %s: %w", manifestPath, err)
	}
	return os.WriteFile(manifestPath, out, 0o644)
}

// AddWorldOpts configures AddWorld. All fields are optional except
// the world name passed as a separate argument.
type AddWorldOpts struct {
	// Agents is the list of agent names this world deploys. Empty is
	// allowed at scaffold time but `spwn check` will flag it.
	Agents []string
	// Workspaces is the list of workspace mount specs. Empty defaults
	// to ["."] so the project root is mounted at /workspace.
	Workspaces []string
}

// AddWorld inserts a new entry under spwn.yaml#worlds with the given
// name and options. Idempotent: a no-op if an entry with that name
// already exists. Used by `spwn world create <name>`.
func AddWorld(manifestPath, name string, opts AddWorldOpts) error {
	data, err := os.ReadFile(manifestPath)
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
	return os.WriteFile(manifestPath, out, 0o644)
}

// RemoveWorld drops the named entry from spwn.yaml#worlds. Returns
// an error wrapping ErrWorldNotFound if no such entry exists. Used
// by `spwn world rm <name>`.
func RemoveWorld(manifestPath, name string) error {
	data, err := os.ReadFile(manifestPath)
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
			return os.WriteFile(manifestPath, out, 0o644)
		}
	}
	return ErrWorldNotFound
}

// ErrWorldNotFound is returned by RemoveWorld when the named world
// does not exist in spwn.yaml.
var ErrWorldNotFound = fmt.Errorf("world not found in spwn.yaml")

// RemoveAgentFromManifest strips every reference to the named agent
// from spwn.yaml#worlds. If a world ends up with zero agents, the
// whole world entry is dropped so `spwn check` doesn't complain
// about an empty agents list. Idempotent: a no-op if the agent is
// not referenced anywhere.
//
// Used by `spwn agent rm` to keep the manifest consistent with disk
// state (symmetric with AddAgentWorld which `agent create` uses).
func RemoveAgentFromManifest(manifestPath, agentName string) error {
	data, err := os.ReadFile(manifestPath)
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
		return nil
	}

	changed := false
	// Walk worlds in reverse so we can delete empty entries in place.
	for i := len(worlds.Content) - 2; i >= 0; i -= 2 {
		entry := worlds.Content[i+1]
		if entry.Kind != yaml.MappingNode {
			continue
		}
		agents := findMapValue(entry, "agents")
		if agents == nil || agents.Kind != yaml.SequenceNode {
			continue
		}
		kept := agents.Content[:0]
		for _, n := range agents.Content {
			if n.Value == agentName {
				changed = true
				continue
			}
			kept = append(kept, n)
		}
		agents.Content = kept
		if len(agents.Content) == 0 {
			// Drop the whole world entry (key + value) — leaving a
			// world with zero agents would be rejected by
			// ruleWorldNames anyway.
			worlds.Content = append(worlds.Content[:i], worlds.Content[i+2:]...)
			changed = true
		}
	}

	if !changed {
		return nil
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("re-encode %s: %w", manifestPath, err)
	}
	return os.WriteFile(manifestPath, out, 0o644)
}

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
	Name string
}

func writeTemplate(root, srcRel, dstRel string, data templateData) error {
	raw, err := templatesFS.ReadFile(srcRel)
	if err != nil {
		return fmt.Errorf("load template %s: %w", srcRel, err)
	}
	tmpl, err := template.New(srcRel).Parse(string(raw))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", srcRel, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render template %s: %w", srcRel, err)
	}
	dst := filepath.Join(root, dstRel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir for %s: %w", dstRel, err)
	}
	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dstRel, err)
	}
	return nil
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

func appendGitignore(root string) error {
	path := filepath.Join(root, ".gitignore")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	needle := ".spwn/"
	if bytes.Contains(existing, []byte("\n"+needle)) || bytes.HasPrefix(existing, []byte(needle+"\n")) || bytes.Equal(existing, []byte(needle)) {
		return nil
	}
	var out bytes.Buffer
	out.Write(existing)
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		out.WriteByte('\n')
	}
	if len(existing) > 0 {
		out.WriteByte('\n')
		out.WriteString("# spwn local state\n")
	}
	out.WriteString(needle)
	out.WriteByte('\n')
	return os.WriteFile(path, out.Bytes(), 0o644)
}
