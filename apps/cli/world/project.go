package world

import (
	"fmt"
	"os"
	"sort"

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
}

// loadProject walks up from the cwd looking for spwn.yaml. Returns
// (nil, nil) when no project is active so callers can fall back to
// the legacy global-mode flow.
func loadProject() (*project.Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return project.Find(cwd)
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

	// Union: explicit world.Packages + every agent.yaml.Packages.
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
	for _, t := range w.Packages {
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
		for _, t := range am.Packages {
			add(t)
		}
	}

	m := world.Manifest{Packages: pkgs}

	return &projectWorld{
		Project:    p,
		Name:       name,
		Agents:     append([]string(nil), w.Agents...),
		Workspaces: append([]string(nil), w.Workspaces...),
		Manifest:   m,
	}, nil
}
