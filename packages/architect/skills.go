package architect

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/dependency/tool"
	"spwn.sh/packages/transpile"
)

// loadRuntimeHooks parses <root>/spwn/hooks.yaml into transpile.HookEntry
// slices for the architect-driven spawn path. Mirrors the source
// package's loadHooks but lives here so the spawn flow doesn't need to
// construct a full ProjectSource just to read the manifest. Malformed
// YAML returns nil rather than erroring — `spwn check` is the
// authoring-side gate; spawn is best-effort.
func loadRuntimeHooks(projectRoot string) []transpile.HookEntry {
	if projectRoot == "" {
		return nil
	}
	path := filepath.Join(projectRoot, "spwn", "hooks.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var parsed struct {
		Hooks []struct {
			Name    string `yaml:"name"`
			Event   string `yaml:"event"`
			Matcher string `yaml:"matcher,omitempty"`
			Command string `yaml:"command"`
		} `yaml:"hooks"`
	}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil
	}
	out := make([]transpile.HookEntry, 0, len(parsed.Hooks))
	for _, h := range parsed.Hooks {
		if h.Name == "" || h.Event == "" || h.Command == "" {
			continue
		}
		out = append(out, transpile.HookEntry{
			Name:    h.Name,
			Event:   h.Event,
			Matcher: h.Matcher,
			Command: h.Command,
		})
	}
	return out
}

// collectRuntimeSkills assembles every skill the compiled tree should
// emit into each agent's native skill directory (.claude/skills/ for
// claude-code, .agents/skills/ for codex). Two sources feed it:
//
//  1. User-authored skills under <projectRoot>/spwn/skills/<name>/
//     (SKILL.md + optional sidecar files) or legacy bare-markdown
//     <projectRoot>/spwn/skills/<name>.md.
//  2. Tool-shipped skills exposed by each resolved tool's Skills()
//     fs.FS — each top-level sub-directory containing SKILL.md is
//     one skill; a Skills() root with SKILL.md at its top is treated
//     as a single skill named after the tool.
//
// Both flow into the same []SkillEntry slice so the renderer sees one
// uniform shape. Tool name collisions with user skills are left
// intact — later-registered entries overwrite earlier ones (tools
// last, users first), which matches today's precedence where the
// user's project spec is explicit.
func collectRuntimeSkills(projectRoot string, tools []tool.Tool) []transpile.SkillEntry {
	out := make([]transpile.SkillEntry, 0)
	out = append(out, loadUserSkills(projectRoot)...)
	out = append(out, toolSkillEntries(tools)...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// loadUserSkills walks <root>/spwn/skills/ following the same rules as
// packages/transpile/source.loadSkills. Kept inline here (not a call
// into source) because the spawn path doesn't go through ProjectSource
// — it constructs transpile.Input manually. Missing project root or
// missing skills dir both return nil.
func loadUserSkills(projectRoot string) []transpile.SkillEntry {
	if projectRoot == "" {
		return nil
	}
	base := filepath.Join(projectRoot, "spwn", "skills")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []transpile.SkillEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			if skill, ok := loadUserSkillDir(filepath.Join(base, name), name); ok {
				out = append(out, skill)
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(base, name))
		if readErr != nil {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		out = append(out, transpile.SkillEntry{
			Name:  id,
			Files: map[string][]byte{"SKILL.md": ensureSkillFrontmatter(body, id)},
		})
	}
	return out
}

// loadUserSkillDir reads a directory-form skill into a SkillEntry.
// Returns ok=false (without error) when SKILL.md is missing — the
// architect tolerates half-authored skills rather than failing the
// whole spawn; spwn check is the authoring-side gate.
func loadUserSkillDir(dir, name string) (transpile.SkillEntry, bool) {
	files := map[string][]byte{}
	err := filepath.Walk(dir, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		rel, relErr := filepath.Rel(dir, p)
		if relErr != nil {
			return relErr
		}
		body, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		files[filepath.ToSlash(rel)] = body
		return nil
	})
	if err != nil {
		return transpile.SkillEntry{}, false
	}
	if _, ok := files["SKILL.md"]; !ok {
		return transpile.SkillEntry{}, false
	}
	files["SKILL.md"] = ensureSkillFrontmatter(files["SKILL.md"], name)
	return transpile.SkillEntry{Name: name, Files: files}, true
}

// toolSkillEntries converts every resolved tool's Skills() fs.FS into
// one or more SkillEntry values. Shape detection:
//
//   - Skills() nil → no contribution.
//   - SKILL.md at the root of the tool's FS → one SkillEntry whose
//     name equals the tool name (stripped of the leading "@").
//   - Otherwise → one SkillEntry per top-level sub-directory that
//     contains a SKILL.md (sub-directories without SKILL.md are
//     ignored — the tool can ship auxiliary assets without them
//     being mistaken for skills).
func toolSkillEntries(tools []tool.Tool) []transpile.SkillEntry {
	var out []transpile.SkillEntry
	for _, t := range tools {
		skillFS := t.Skills()
		if skillFS == nil {
			continue
		}
		// Strip the dependency scheme prefix so the skill directory
		// matches the bare skill slug (`qmd`) rather than the dep ref
		// (`spwn:qmd`). Colons in path components are legal on Linux
		// but break plenty of downstream tooling — and the SKILL.md's
		// own `name:` frontmatter uses the bare slug, so the dir name
		// should agree.
		toolName := skillDirName(t.Name())

		// Flat form: SKILL.md at FS root.
		if rootBody, err := fs.ReadFile(skillFS, "SKILL.md"); err == nil {
			files := map[string][]byte{"SKILL.md": rootBody}
			collectSidecar(skillFS, ".", files)
			out = append(out, transpile.SkillEntry{Name: toolName, Files: files})
			continue
		}

		// Nested form: each top-level sub-directory with SKILL.md.
		topLevel, err := fs.ReadDir(skillFS, ".")
		if err != nil {
			continue
		}
		for _, entry := range topLevel {
			if !entry.IsDir() {
				continue
			}
			sub, subErr := fs.Sub(skillFS, entry.Name())
			if subErr != nil {
				continue
			}
			body, err := fs.ReadFile(sub, "SKILL.md")
			if err != nil {
				continue
			}
			files := map[string][]byte{"SKILL.md": body}
			collectSidecar(sub, ".", files)
			out = append(out, transpile.SkillEntry{Name: entry.Name(), Files: files})
		}
	}
	return out
}

// collectSidecar walks an fs.FS rooted at `root` and records every
// non-SKILL.md file into `out`, keyed by path relative to the root.
// Used so tools that ship templates / scripts / references alongside
// their SKILL.md keep those files together in the rendered output.
func collectSidecar(fsys fs.FS, root string, out map[string][]byte) {
	_ = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if p == "SKILL.md" {
			return nil
		}
		// Normalise root == "." → clean relative path.
		rel := p
		if root != "." && root != "" {
			rel = strings.TrimPrefix(p, root+"/")
		}
		body, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return nil
		}
		out[filepath.ToSlash(rel)] = body
		return nil
	})
}

// ensureSkillFrontmatter matches the loader's helper in
// packages/transpile/source.  Duplicated (not imported) so the
// architect package doesn't pull in the source loader just for this
// four-line shim.
func ensureSkillFrontmatter(body []byte, name string) []byte {
	trimmed := strings.TrimLeft(string(body), "\n\t ")
	if strings.HasPrefix(trimmed, "---") {
		return body
	}
	header := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n", name, name)
	return append([]byte(header), body...)
}

// skillDirName turns a dep ref into the directory component used under
// `.claude/skills/`. The goal is the bare slug: `spwn:qmd` → `qmd`,
// `local:my-skill` → `my-skill` (post-hydrate registry key),
// `tool/foo` → `foo`, `skill/bar` → `bar`, `hook/baz` → `baz`. Keep
// in lockstep with refs.Kind recognisers in packages/dependency/refs.
func skillDirName(ref string) string {
	// Internal post-hydrate registry keys (`spwn:`, `local:`) are not
	// user-facing and don't go through the parser.
	for _, scheme := range []string{"spwn:", "local:"} {
		if strings.HasPrefix(ref, scheme) {
			return strings.TrimPrefix(ref, scheme)
		}
	}
	// User-facing path-style local refs (tool/foo, skill/bar, hook/baz).
	for _, prefix := range []string{"tool/", "skill/", "hook/"} {
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimPrefix(ref, prefix)
		}
	}
	return strings.TrimPrefix(ref, "@")
}
