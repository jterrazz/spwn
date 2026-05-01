package source

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"spwn.sh/packages/transpile"
)

// ToCompileInput projects a ProjectSource onto the transpile.Input shape
// that transpile.Runtime.Render expects. worldName selects which world
// from the manifest to compile; pass an empty string and ToCompileInput
// will pick "the only world" (error if multiple worlds exist).
//
// Tool verification is skipped here — callers feeding the result
// through a real spawn can override VerifiedTools; for CLI dry-runs
// the union of agent tools is the best we can do without probing a
// container.
func ToCompileInput(src *ProjectSource, worldName string) (transpile.Input, error) {
	if src == nil {
		return transpile.Input{}, fmt.Errorf("nil ProjectSource")
	}
	if src.Manifest == nil {
		return transpile.Input{}, fmt.Errorf("ProjectSource has no manifest")
	}

	worlds := src.Manifest.Worlds
	if len(worlds) == 0 {
		return transpile.Input{}, fmt.Errorf("no worlds declared in spwn.yaml")
	}

	selected := worldName
	if selected == "" {
		switch len(worlds) {
		case 1:
			for k := range worlds {
				selected = k
			}
		default:
			names := make([]string, 0, len(worlds))
			for k := range worlds {
				names = append(names, k)
			}
			sort.Strings(names)
			return transpile.Input{}, fmt.Errorf(
				"manifest declares multiple worlds (%v); pick one with --world",
				names,
			)
		}
	}

	world, ok := worlds[selected]
	if !ok {
		names := make([]string, 0, len(worlds))
		for k := range worlds {
			names = append(names, k)
		}
		sort.Strings(names)
		return transpile.Input{}, fmt.Errorf("world %q not in manifest (have: %v)", selected, names)
	}

	// Index agents on disk by name.
	byName := make(map[string]AgentSource, len(src.Agents))
	for _, a := range src.Agents {
		byName[a.Name] = a
	}

	// Index project hooks by name so we can attach per-agent slices
	// based on each agent's `hook/<n>` deps. Hooks an agent doesn't
	// declare are silently absent for that agent — selection is
	// explicit, mirroring how skill/<name> and tool/<name> work.
	hookByName := make(map[string]transpile.HookEntry, len(src.Hooks))
	for _, h := range src.Hooks {
		hookByName[h.Name] = transpile.HookEntry{
			Name:    h.Name,
			Event:   h.Event,
			Matcher: h.Matcher,
			Command: h.Command,
		}
	}

	// Same pattern for slash-invoked commands.
	commandByName := make(map[string]transpile.CommandEntry, len(src.Commands))
	for _, c := range src.Commands {
		commandByName[c.Name] = transpile.CommandEntry{
			Name: c.Name,
			Body: c.Body,
		}
	}

	// Collect the union of packages from every agent in this world.
	// This mirrors what spawn does before probing the container: the
	// render doesn't need a verified list, it just needs to know what
	// the manifest *claims* is available.
	packages := map[string]struct{}{}
	agents := make([]transpile.AgentInput, 0, len(world.Agents))
	for _, name := range world.Agents {
		a, ok := byName[name]
		if !ok {
			return transpile.Input{}, fmt.Errorf(
				"world %q references missing agent %q", selected, name)
		}
		for _, p := range a.Config.Deps {
			packages[p] = struct{}{}
		}
		agents = append(agents, transpile.AgentInput{
			Name:      a.Name,
			Role:      a.Config.Role,
			Soul:      a.Soul,
			AgentMD:   a.AgentMD,
			Playbooks: promotedPlaybooks(a.Layers.Playbooks),
			Model:     a.Config.Runtime.Model,
			Provider:  a.Config.Runtime.Provider,
			Hooks:     selectAgentHooks(a.Config.Deps, hookByName),
			Commands:  selectAgentCommands(a.Config.Deps, commandByName),
		})
	}
	// Add project-level deps (top-level deps: in spwn.yaml).
	for _, p := range src.Manifest.Deps {
		packages[p] = struct{}{}
	}

	packageList := make([]string, 0, len(packages))
	for p := range packages {
		packageList = append(packageList, p)
	}
	sort.Strings(packageList)

	// WorldKnowledgeMounted mirrors what the spawn pipeline would do
	// at runtime: when the world declares a knowledge path AND the
	// directory exists on disk, the renderer emits the "with
	// knowledge" boilerplate. Empty key or missing dir → omit every
	// /world/knowledge/ reference so the agent is never told a
	// knowledge base exists. This matches the architect's behaviour
	// in packages/architect/spawn.go.
	knowledgeMounted := false
	if kp := strings.TrimSpace(world.Knowledge); kp != "" {
		resolved := kp
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(src.RootDir, kp)
		}
		if info, err := os.Stat(resolved); err == nil && info.IsDir() {
			knowledgeMounted = true
		}
	}

	skills := make([]transpile.SkillEntry, 0, len(src.Skills))
	for _, s := range src.Skills {
		skills = append(skills, transpile.SkillEntry{
			Name:  s.Name,
			Files: s.Files,
		})
	}

	hooks := make([]transpile.HookEntry, 0, len(src.Hooks))
	for _, h := range src.Hooks {
		hooks = append(hooks, transpile.HookEntry{
			Name:    h.Name,
			Event:   h.Event,
			Matcher: h.Matcher,
			Command: h.Command,
		})
	}

	commands := make([]transpile.CommandEntry, 0, len(src.Commands))
	for _, c := range src.Commands {
		commands = append(commands, transpile.CommandEntry{
			Name: c.Name,
			Body: c.Body,
		})
	}

	return transpile.Input{
		Deps:                  packageList,
		VerifiedTools:         packageList,
		WorldID:               selected,
		Agents:                agents,
		WorldKnowledgeMounted: knowledgeMounted,
		Skills:                skills,
		Hooks:                 hooks,
		Commands:              commands,
	}, nil
}

// selectAgentCommands returns the subset of project commands the
// agent's dependency list explicitly subscribes to via `command/<n>`
// refs. Mirrors selectAgentHooks; unknown command names are dropped
// silently here and surface as resolver errors via the validator.
func selectAgentCommands(deps []string, byName map[string]transpile.CommandEntry) []transpile.CommandEntry {
	if len(deps) == 0 || len(byName) == 0 {
		return nil
	}
	out := make([]transpile.CommandEntry, 0, len(deps))
	seen := make(map[string]struct{}, len(deps))
	for _, dep := range deps {
		ref := strings.TrimSpace(dep)
		const prefix = "command/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}
		name := strings.TrimPrefix(ref, prefix)
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

// selectAgentHooks returns the subset of project hooks the agent's
// dependency list explicitly subscribes to via `hook/<n>` refs.
// Unknown hook names (declared in agent.yaml but not present under
// spwn/hooks/) are silently dropped here — the validator surfaces
// those as proper errors via the ResolveTool path.
func selectAgentHooks(deps []string, byName map[string]transpile.HookEntry) []transpile.HookEntry {
	if len(deps) == 0 || len(byName) == 0 {
		return nil
	}
	out := make([]transpile.HookEntry, 0, len(deps))
	seen := make(map[string]struct{}, len(deps))
	for _, dep := range deps {
		ref := strings.TrimSpace(dep)
		const prefix = "hook/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}
		name := strings.TrimPrefix(ref, prefix)
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

// WorldNames returns the set of world names declared in the manifest,
// sorted alphabetically. Returns nil when src or its manifest is nil.
func WorldNames(src *ProjectSource) []string {
	if src == nil || src.Manifest == nil {
		return nil
	}
	out := make([]string, 0, len(src.Manifest.Worlds))
	for k := range src.Manifest.Worlds {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
