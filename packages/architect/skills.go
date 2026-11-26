package architect

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/dependency/tool"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/transpile"
)

func loadRuntimeHooks(project *platform.Tree) []transpile.HookEntry {
	if project == nil {
		return nil
	}
	dir := filepath.Join("spwn", "hooks")
	entries, err := project.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]transpile.HookEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		data, rerr := project.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			continue
		}
		var parsed struct {
			Event   string `yaml:"event"`
			Matcher string `yaml:"matcher,omitempty"`
			Command string `yaml:"command"`
		}
		if uerr := yaml.Unmarshal(data, &parsed); uerr != nil {
			continue
		}
		if parsed.Event == "" || parsed.Command == "" {
			continue
		}
		out = append(out, transpile.HookEntry{
			Name:    name,
			Event:   parsed.Event,
			Matcher: parsed.Matcher,
			Command: parsed.Command,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func agentHookSelection(agentName string, pool []transpile.HookEntry) []transpile.HookEntry {
	if len(pool) == 0 {
		return nil
	}
	m, err := agent.LoadManifest(agentName)
	if err != nil || m == nil || len(m.Deps) == 0 {
		return nil
	}
	byName := make(map[string]transpile.HookEntry, len(pool))
	for _, h := range pool {
		byName[h.Name] = h
	}
	out := make([]transpile.HookEntry, 0, len(m.Deps))
	seen := make(map[string]struct{}, len(m.Deps))
	for _, dep := range m.Deps {
		dep = strings.TrimSpace(dep)
		const prefix = "hook/"
		if !strings.HasPrefix(dep, prefix) {
			continue
		}
		name := strings.TrimPrefix(dep, prefix)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		entry, ok := byName[name]
		if !ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func loadRuntimeCommands(project *platform.Tree) []transpile.CommandEntry {
	if project == nil {
		return nil
	}
	dir := filepath.Join("spwn", "commands")
	entries, err := project.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]transpile.CommandEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			continue
		}
		body, rerr := project.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			continue
		}
		out = append(out, transpile.CommandEntry{
			Name: strings.TrimSuffix(name, ".md"),
			Body: body,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func agentCommandSelection(agentName string, pool []transpile.CommandEntry) []transpile.CommandEntry {
	if len(pool) == 0 {
		return nil
	}
	m, err := agent.LoadManifest(agentName)
	if err != nil || m == nil || len(m.Deps) == 0 {
		return nil
	}
	byName := make(map[string]transpile.CommandEntry, len(pool))
	for _, c := range pool {
		byName[c.Name] = c
	}
	out := make([]transpile.CommandEntry, 0, len(m.Deps))
	seen := make(map[string]struct{}, len(m.Deps))
	for _, dep := range m.Deps {
		dep = strings.TrimSpace(dep)
		const prefix = "command/"
		if !strings.HasPrefix(dep, prefix) {
			continue
		}
		name := strings.TrimPrefix(dep, prefix)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		entry, ok := byName[name]
		if !ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func collectRuntimeSkills(project *platform.Tree, tools []tool.Tool) []transpile.SkillEntry {
	out := make([]transpile.SkillEntry, 0)
	out = append(out, loadUserSkills(project)...)
	out = append(out, toolSkillEntries(tools)...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func facultiesForRuntime(verifiedTools, deps []string) []string {
	out := make([]string, 0, len(verifiedTools)+len(deps))
	seen := map[string]struct{}{}
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, ok := seen[ref]; ok {
			return
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	for _, ref := range verifiedTools {
		add(ref)
	}
	localSkills := make([]string, 0)
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if strings.HasPrefix(dep, "skill/") {
			localSkills = append(localSkills, dep)
		}
	}
	sort.Strings(localSkills)
	for _, ref := range localSkills {
		add(ref)
	}
	return out
}

func loadUserSkills(project *platform.Tree) []transpile.SkillEntry {
	if project == nil {
		return nil
	}
	base := filepath.Join("spwn", "skills")
	entries, err := project.ReadDir(base)
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
			if skill, ok := loadUserSkillDir(project, filepath.Join(base, name), name); ok {
				out = append(out, skill)
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		body, readErr := project.ReadFile(filepath.Join(base, name))
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

func loadUserSkillDir(project *platform.Tree, dir, name string) (transpile.SkillEntry, bool) {
	files := map[string][]byte{}
	err := project.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		rel, relErr := filepath.Rel(dir, p)
		if relErr != nil {
			return relErr
		}
		body, readErr := project.ReadFile(p)
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

func toolSkillEntries(tools []tool.Tool) []transpile.SkillEntry {
	var out []transpile.SkillEntry
	for _, t := range tools {
		skillFS := t.Skills()
		if skillFS == nil {
			continue
		}
		toolName := skillDirName(t.Name())

		if rootBody, err := fs.ReadFile(skillFS, "SKILL.md"); err == nil {
			files := map[string][]byte{"SKILL.md": rootBody}
			collectSidecar(skillFS, ".", files)
			out = append(out, transpile.SkillEntry{Name: toolName, Files: files})
			continue
		}

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

func collectSidecar(fsys fs.FS, root string, out map[string][]byte) {
	_ = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if p == "SKILL.md" {
			return nil
		}
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

func ensureSkillFrontmatter(body []byte, name string) []byte {
	trimmed := strings.TrimLeft(string(body), "\n\t ")
	if strings.HasPrefix(trimmed, "---") {
		return body
	}
	header := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n", name, name)
	return append([]byte(header), body...)
}

func skillDirName(ref string) string {
	for _, scheme := range []string{"spwn:", "local:"} {
		if strings.HasPrefix(ref, scheme) {
			return strings.TrimPrefix(ref, scheme)
		}
	}
	for _, prefix := range []string{"tool/", "skill/", "hook/"} {
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimPrefix(ref, prefix)
		}
	}
	return strings.TrimPrefix(ref, "@")
}
