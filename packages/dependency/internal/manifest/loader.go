package manifest

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest is the canonical basename. Both catalog and local
// dependencies live at <depDir>/spwn.yaml.
// Manifest is the basename of the project-level manifest that
// defines a spwn project (name, worlds, deps, lockfile anchor).
const Manifest = "spwn.yaml"

// ToolManifest is the basename for an individual tool definition
// (install / verify / files / skills-sibling).
// Lives under spwn/tools/<name>/tool.yaml in a user project and
// catalog/<slug>/tools/tool.yaml in the catalog.
const ToolManifest = "tool.yaml"

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
	// Catalog loaders pass "spwn:<basename>"; the project-local
	// loader passes the bare directory name.
	DefaultName string

	// DefaultVersion is applied when the manifest doesn't set
	// `version:`. Project-local tools default to "0.0.0-local";
	// catalog tools default to "latest".
	DefaultVersion string

	// ManifestFile overrides the on-disk filename the resolver
	// reads. Empty defaults to Manifest ("spwn.yaml"). Callers
	// reading tool-specific manifests pass ToolManifest
	// ("tool.yaml"). Error messages and unmarshal context carry
	// this filename so debugging stays clear.
	ManifestFile string
}

// Parsed is the result of parsing a spwn.yaml dependency manifest.
// It carries the schema plus any files read eagerly from the resolver
// so the result can outlive the Resolver. Converters in other packages
// (e.g. dependency.ToolFromParsed) adapt this into their own types.
type Parsed struct {
	Schema    Schema
	FileBytes map[string][]byte
	SkillsFS  any // fs.FS but typed as any to avoid the import on the public type
}

// Parse reads the manifest file via the Resolver and returns a
// Parsed. File references declared in the `files:` map are read
// eagerly. The manifest filename defaults to "spwn.yaml"; pass
// ParseOptions.ManifestFile = ToolManifest to read "tool.yaml"
// for tool-level manifests.
func Parse(res Resolver, opts ParseOptions) (*Parsed, error) {
	manifestFile := opts.ManifestFile
	if manifestFile == "" {
		manifestFile = Manifest
	}
	data, err := res.ReadFile(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", manifestFile, err)
	}

	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifestFile, err)
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

	return &Parsed{
		Schema:    s,
		FileBytes: fileBytes,
		SkillsFS:  res.SkillsFS(),
	}, nil
}

// DirResolver is a Resolver backed by a host filesystem directory.
// Used by the project-local tool loader for spwn/tools/<name>/.
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
// catalog loader so dependency manifests ship inside the spwn binary.
//
// Example:
//
//	//go:embed all:content
//	var catalogFS embed.FS
//
//	res := EmbedResolver{FS: catalogFS, Root: "content/git/tools"}
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

