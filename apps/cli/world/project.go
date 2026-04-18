package world

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/packages/project"
	"spwn.sh/packages/world"
)

// projectWorld is the fully-resolved spawn plan for one inline world
// entry in spwn.yaml. It contains everything spawnRunE needs to skip
// the legacy ~/.spwn/worlds/<name>.yaml file load entirely.
type projectWorld struct {
	Project    *project.Project
	Name       string
	Agents     []string
	Workspaces []string
	Manifest   world.Manifest
	// Knowledge is the absolute host path to bind into
	// /world/knowledge/, resolved from the manifest's
	// worlds.<name>.knowledge key relative to the project root.
	// Empty when the manifest declares no knowledge path — in which
	// case the spawn pipeline skips the bind mount entirely and the
	// rendered AGENTS.md / CLAUDE.md / mind-management skill omit
	// every reference to /world/knowledge/.
	Knowledge string
}

// loadProject is kept as a thin alias over cliproject.Find so the
// existing call sites under apps/cli/world/ stay concise while the
// canonical walker lives in cliproject.
func loadProject() (*project.Project, error) {
	return cliproject.Find()
}

// sortedWorldNames returns the project's world-map keys in stable
// alphabetic order so callers iterate predictably.
func sortedWorldNames(p *project.Project) []string {
	if p == nil || p.Manifest == nil {
		return nil
	}
	names := make([]string, 0, len(p.Manifest.Worlds))
	for name := range p.Manifest.Worlds {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resolveProjectWorld synthesizes a projectWorld for the named world
// entry. If name is empty, the first entry (sorted) is used. Returns
// an error when the requested world does not exist.
//
// The resulting world.Manifest has Packages unioned from (1) the
// inline world's explicit Packages list and (2) every referenced
// agent's agent.yaml Packages field.
func resolveProjectWorld(p *project.Project, name string) (*projectWorld, error) {
	if p == nil || p.Manifest == nil {
		return nil, fmt.Errorf("no project loaded")
	}
	if len(p.Manifest.Worlds) == 0 {
		return nil, fmt.Errorf("spwn.yaml has no worlds")
	}
	if name == "" {
		// Pick the first sorted entry as the implicit default.
		name = sortedWorldNames(p)[0]
	}
	w, ok := p.Manifest.Worlds[name]
	if !ok {
		return nil, fmt.Errorf("world %q is not defined in spwn.yaml", name)
	}

	// Union: project-level deps + every agent.yaml dependency.
	pkgSet := map[string]struct{}{}
	var pkgs []string
	add := func(t string) {
		if t == "" {
			return
		}
		if _, seen := pkgSet[t]; seen {
			return
		}
		pkgSet[t] = struct{}{}
		pkgs = append(pkgs, t)
	}
	for _, t := range p.Manifest.Deps {
		add(t)
	}
	// p.Agents is the deployable set. We only want the subset referenced
	// by this particular world entry, so re-resolve by name.
	agentPath := map[string]string{}
	for _, a := range p.Agents {
		agentPath[a.Name] = a.Path
	}
	for _, a := range p.OrphanAgents {
		agentPath[a.Name] = a.Path
	}
	for _, aname := range w.Agents {
		dir, ok := agentPath[aname]
		if !ok {
			continue
		}
		am, err := world.LoadAgentManifest(dir)
		if err != nil || am == nil {
			continue
		}
		for _, t := range am.Deps {
			add(t)
		}
	}

	m := world.Manifest{Deps: pkgs}

	// Resolve the knowledge path (if any) relative to the project root
	// so SpawnOpts receives an absolute host path. Empty string means
	// "no knowledge base" — the spawn pipeline drops the bind mount and
	// compile omits every /world/knowledge/ reference.
	var knowledge string
	if kp := strings.TrimSpace(w.Knowledge); kp != "" {
		if filepath.IsAbs(kp) {
			knowledge = kp
		} else {
			knowledge = filepath.Join(p.Root, kp)
		}
	}

	return &projectWorld{
		Project:    p,
		Name:       name,
		Agents:     append([]string(nil), w.Agents...),
		Workspaces: append([]string(nil), w.Workspaces...),
		Manifest:   m,
		Knowledge:  knowledge,
	}, nil
}
