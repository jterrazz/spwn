package manifest

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is the canonical basename. Both catalog and local
// dependencies live at <depDir>/spwn.yaml.
// Manifest is the basename of the project-level manifest that
// defines a spwn project (name, worlds, deps, lockfile anchor).
const Manifest = "spwn.yaml"

// ToolManifest is the basename for an individual tool definition
// (install / verify / files). Lives under tools/<name>/tool.yaml
// in a user project and catalog/<slug>/tools/<name>/tool.yaml in the
// catalog.
const ToolManifest = "tool.yaml"

// Resolver handles filesystem lookups for a tool's manifest and
// supporting directories (files/, skills/, config/). It abstracts
// over "read from disk" (local tools) vs "read from go:embed"
// (catalog tools) so both paths share the parser.
type Resolver interface {
	ReadFile(rel string) ([]byte, error)

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
	SkillsFS  any
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

	fileBytes := make(map[string][]byte, len(s.Files))
	for imagePath, sourcePath := range s.Files {
		if leavesTheFolder(sourcePath) {
			return nil, fmt.Errorf("%s: files[%q]: source %q %w", s.Name, imagePath, sourcePath, ErrLeavesTheTool)
		}
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

// ErrLeavesTheTool is the error of a `files:` source that names a path
// outside the tool's folder: by "..", by an absolute path or by a NUL.
// It is refused by its name alone, before anything is read.
var ErrLeavesTheTool = errors.New("leaves the tool's folder")

func leavesTheFolder(source string) bool {
	return source != "" && (strings.ContainsRune(source, 0) || !filepath.IsLocal(filepath.FromSlash(source)))
}

// FSResolver is a Resolver backed by an fs.FS: the embedded catalog's,
// so dependency manifests ship inside the spwn binary, or a project's
// folder read through its root (platform.FS), for the tools under
// tools/<name>/.
//
// Example:
//
//	//go:embed all:content
//	var catalogFS embed.FS
//
//	res := FSResolver{
//	    FS:         catalogFS,
//	    Root:       "content/git/tools/git",  // manifest + files:
//	    SkillsRoot: "content/git/skills",      // catalog-entry-level
//	}
//
// Paths inside an FSResolver always use forward slashes, as fs.FS
// names do on every host.
type FSResolver struct {
	FS   fs.FS
	Root string

	// SkillsRoot overrides the location of the skills/ directory.
	// Empty defaults to <Root>/skills. Catalog entries set this to
	// the catalog-entry root so skills live alongside tools/ rather
	// than buried under a specific tool's dir.
	SkillsRoot string
}

// ReadFile reads <Root>/<rel> from the FS.
func (e FSResolver) ReadFile(rel string) ([]byte, error) {
	return fs.ReadFile(e.FS, path.Join(e.Root, rel))
}

// SkillsFS returns an fs.Sub of the configured skills directory when
// it exists and contains at least one entry. Uses SkillsRoot when
// set, otherwise falls back to <Root>/skills.
func (e FSResolver) SkillsFS() fs.FS {
	skillsRoot := e.SkillsRoot
	if skillsRoot == "" {
		skillsRoot = path.Join(e.Root, "skills")
	}
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
