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
		{"templates/world.yaml.tmpl", "spwn/worlds/default.yaml"},
		{"templates/agent.yaml.tmpl", "spwn/agents/neo/agent.yaml"},
		{"templates/CLAUDE.md.tmpl", "spwn/agents/neo/CLAUDE.md"},
		{"templates/profile.md.tmpl", "spwn/agents/neo/core/profile.md"},
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

	// Local state dir (.spwn/) is gitignored but needs to exist so
	// tools that read state.json don't have to branch on missing.
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
	// slug-ify: keep alnum + dash, lowercase the rest
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

// appendGitignore adds .spwn/ to the existing .gitignore, or creates
// a fresh one if none exists. Idempotent: won't duplicate the line.
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
