package architect

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	ib "spwn.sh/packages/image"
)

// localToolYAML is the schema for spwn/tools/<name>/package.yaml —
// the manifest for a project-local tool pack. Kept intentionally
// small so authoring a local tool is "fill in three keys". Unknown
// fields are ignored for forwards compatibility.
type localToolYAML struct {
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Description  string            `yaml:"description"`
	Dependencies []string          `yaml:"dependencies"`
	Packages     []string          `yaml:"packages"`
	Commands     []string          `yaml:"commands"`
	UserCommands []string          `yaml:"user-commands"`
	Env          map[string]string `yaml:"env"`
	Verify       []string          `yaml:"verify"`
}

// localTool is the image.Tool adapter for a project-local pack. It's
// backed by the parsed package.yaml plus an optional skills/ dir
// exposed through Skills(). Name() returns the "local:<basename>"
// form so the synthetic entry doesn't collide with any future
// @spwn/<name> promotion.
type localTool struct {
	name     string // "local:<basename>"
	version  string
	deps     []string
	spec     ib.InstallSpec
	verify   []string
	skillsFS fs.FS
}

func (t *localTool) Name() string          { return t.name }
func (t *localTool) Kind() ib.Kind         { return ib.KindTool }
func (t *localTool) Version() string       { return t.version }
func (t *localTool) Dependencies() []string { return t.deps }
func (t *localTool) Install() ib.InstallSpec { return t.spec }
func (t *localTool) Verify() []string      { return t.verify }
func (t *localTool) Skills() fs.FS         { return t.skillsFS }

// Runtimes and Config satisfy the image.Tool interface for the
// plugin-config pathway. Local packages don't currently declare a
// plugin: section in spwn-tool.yaml (no schema for it yet in the
// local loader) so they always return nil, which the spawn-time
// merger correctly ignores.
func (t *localTool) Runtimes() []string         { return nil }
func (t *localTool) Config(runtime string) []byte { return nil }

// loadLocalTool parses spwn/tools/<name>/package.yaml and produces
// an image.Tool. Missing manifest is a crisp authoring error — we
// don't silently accept empty directories, since an empty local tool
// would render to nothing and the user would spend an afternoon
// debugging a no-op.
func loadLocalTool(projectRoot, name string) (ib.Tool, error) {
	toolDir := filepath.Join(projectRoot, "spwn", "tools", name)
	info, err := os.Stat(toolDir)
	if err != nil {
		return nil, fmt.Errorf("local tool %q: %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local tool %q: %s is not a directory", name, toolDir)
	}

	manifestPath := filepath.Join(toolDir, "package.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("local tool %q: missing package.yaml at %s\n"+
				"  Create one with at least `name:` and `commands:` or `packages:`.",
				name, manifestPath)
		}
		return nil, fmt.Errorf("local tool %q: read manifest: %w", name, err)
	}

	var y localToolYAML
	if err := yaml.Unmarshal(data, &y); err != nil {
		return nil, fmt.Errorf("local tool %q: parse package.yaml: %w", name, err)
	}

	// Optional skills/ subdir. An empty or missing dir → nil so the
	// image builder knows to skip skill collection for this tool.
	var skillsFS fs.FS
	skillsDir := filepath.Join(toolDir, "skills")
	if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
		skillsFS = os.DirFS(skillsDir)
	}

	version := y.Version
	if version == "" {
		version = "0.0.0-local"
	}

	return &localTool{
		name:    "local:" + name,
		version: version,
		deps:    y.Dependencies,
		spec: ib.InstallSpec{
			Packages:     y.Packages,
			Commands:     y.Commands,
			UserCommands: y.UserCommands,
			Env:          y.Env,
		},
		verify:   y.Verify,
		skillsFS: skillsFS,
	}, nil
}

// hydrateLocalTools walks a flat list of tool refs, loads every bare
// (non-@) name as a synthetic image.Tool, registers it with the
// registry, and returns the rewritten list where each bare name has
// been replaced by its "local:<name>" registry key.
//
// Order is preserved so users see their ref list echoed back in the
// same shape they declared it. Duplicates are tolerated — the
// registry's Register is called once per unique name.
func hydrateLocalTools(reg *ib.Registry, projectRoot string, toolList []string) ([]string, error) {
	out := make([]string, 0, len(toolList))
	loaded := map[string]bool{}
	for _, raw := range toolList {
		if strings.HasPrefix(raw, "@") {
			out = append(out, raw)
			continue
		}
		// Bare name — a local tool.
		if loaded[raw] {
			out = append(out, "local:"+raw)
			continue
		}
		tool, err := loadLocalTool(projectRoot, raw)
		if err != nil {
			return nil, err
		}
		if err := reg.Register(tool); err != nil {
			return nil, fmt.Errorf("register local tool %q: %w", raw, err)
		}
		loaded[raw] = true
		out = append(out, "local:"+raw)
	}
	return out, nil
}
