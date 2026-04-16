package packyaml

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	ib "spwn.sh/packages/image"
)

// Manifest is the canonical basename. Both catalog and local
// plugins live at <pluginDir>/plugin.yaml.
const Manifest = "pack.yaml"

// Resolver handles filesystem lookups for a tool's manifest and
// supporting directories (files/, skills/, config/). It abstracts
// over "read from disk" (local tools) vs "read from go:embed"
// (catalog tools) so both paths share the parser.
type Resolver interface {
	// ReadFile returns the bytes of the named file relative to the
	// tool's directory. Returns os.ErrNotExist when absent.
	ReadFile(rel string) ([]byte, error)

	// SkillsFS returns an fs.FS rooted at <toolDir>/skills/, or nil
	// when the directory is absent or empty.
	SkillsFS() fs.FS
}

// ParseOptions configures a Parse call.
type ParseOptions struct {
	// DefaultName is applied when the manifest doesn't set `name:`.
	// Catalog loaders pass "@spwn/<basename>"; the project-local
	// loader passes the bare directory name.
	DefaultName string

	// DefaultVersion is applied when the manifest doesn't set
	// `version:`. Project-local tools default to "0.0.0-local";
	// catalog tools default to "latest".
	DefaultVersion string
}

// Parse reads plugin.yaml via the Resolver and returns an
// image.Tool instance backed by the parsed schema. File references
// declared in the `files:` map are read eagerly — the returned Tool
// is self-contained and can outlive the Resolver.
func Parse(res Resolver, opts ParseOptions) (ib.Tool, error) {
	data, err := res.ReadFile(Manifest)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", Manifest, err)
	}

	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", Manifest, err)
	}

	if s.Name == "" {
		s.Name = opts.DefaultName
	}
	if s.Name == "" {
		return nil, fmt.Errorf("%s: name is required (no default configured)", Manifest)
	}
	if s.Version == "" {
		s.Version = opts.DefaultVersion
	}
	if s.Version == "" {
		s.Version = "latest"
	}
	if s.Kind == "" {
		s.Kind = "tool"
	}

	kind, err := parseKind(s.Kind)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.Name, err)
	}

	// Read every file in the files map eagerly so the returned Tool
	// doesn't depend on the resolver staying alive.
	fileBytes := make(map[string][]byte, len(s.Files))
	for imagePath, sourcePath := range s.Files {
		b, err := res.ReadFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("%s: read file %q: %w", s.Name, sourcePath, err)
		}
		fileBytes[imagePath] = b
	}

	return &toolImpl{
		schema:    s,
		kind:      kind,
		fileBytes: fileBytes,
		skillsFS:  res.SkillsFS(),
	}, nil
}

// DirResolver is a Resolver backed by a host filesystem directory.
// Used by the project-local pack loader for spwn/packs/<name>/.
type DirResolver struct {
	Root string
}

// ReadFile reads <Root>/<rel> from disk.
func (d DirResolver) ReadFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.Root, filepath.FromSlash(rel)))
}

// SkillsFS returns an os.DirFS rooted at <Root>/skills/ when that
// directory exists and contains at least one entry.
func (d DirResolver) SkillsFS() fs.FS {
	skillsDir := filepath.Join(d.Root, "skills")
	info, err := os.Stat(skillsDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(skillsDir)
	if err != nil || len(entries) == 0 {
		return nil
	}
	return os.DirFS(skillsDir)
}

// EmbedResolver is a Resolver backed by an embed.FS. Used by the
// catalog loader so pack manifests ship inside the spwn binary.
//
// Example:
//
//	//go:embed all:catalog/plugins
//	var catalogFS embed.FS
//
//	res := EmbedResolver{FS: catalogFS, Root: "catalog/plugins/git"}
//
// Paths inside EmbedResolver always use forward slashes because
// embed.FS normalises to POSIX regardless of host OS.
type EmbedResolver struct {
	FS   fs.FS
	Root string
}

// ReadFile reads <Root>/<rel> from the embedded filesystem.
func (e EmbedResolver) ReadFile(rel string) ([]byte, error) {
	return fs.ReadFile(e.FS, path.Join(e.Root, rel))
}

// SkillsFS returns an fs.Sub of <Root>/skills/ when that directory
// exists and contains at least one entry.
func (e EmbedResolver) SkillsFS() fs.FS {
	skillsRoot := path.Join(e.Root, "skills")
	entries, err := fs.ReadDir(e.FS, skillsRoot)
	if err != nil || len(entries) == 0 {
		return nil
	}
	sub, err := fs.Sub(e.FS, skillsRoot)
	if err != nil {
		return nil
	}
	return sub
}

func parseKind(s string) (ib.Kind, error) {
	switch strings.ToLower(s) {
	case "runtime":
		return ib.KindRuntime, nil
	case "sdk":
		return ib.KindSDK, nil
	case "tool":
		return ib.KindTool, nil
	case "platform":
		return ib.KindPlatform, nil
	}
	return "", fmt.Errorf("unknown kind %q (want runtime|sdk|tool|platform)", s)
}
