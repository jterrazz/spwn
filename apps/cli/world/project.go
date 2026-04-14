package world

import (
	"fmt"
	"os"
	"sort"

	"spwn.sh/packages/manifest"
	"spwn.sh/packages/world"
)

// projectWorld is the fully-resolved spawn plan for one inline world
// entry in spwn.yaml. It contains everything spawnRunE needs to skip
// the legacy ~/.spwn/worlds/<name>.yaml file load entirely.
type projectWorld struct {
	Project    *manifest.Project
	Name       string
	Agents     []string
	Workspaces []string
	Manifest   world.Manifest
}

// loadProject walks up from the cwd looking for spwn.yaml. Returns
// (nil, nil) when no project is active so callers can fall back to
// the legacy global-mode flow.
func loadProject() (*manifest.Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return manifest.Find(cwd)
}

// sortedWorldNames returns the project's world-map keys in stable
// alphabetic order so callers iterate predictably.
func sortedWorldNames(p *manifest.Project) []string {
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
// The resulting world.Manifest has Tools unioned from (1) the inline
// world's explicit Tools list and (2) every referenced agent's
// agent.yaml Tools field.
func resolveProjectWorld(p *manifest.Project, name string) (*projectWorld, error) {
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

	// Union: explicit world.Tools + every agent.yaml.Tools.
	toolSet := map[string]struct{}{}
	var tools []string
	add := func(t string) {
		if t == "" {
			return
		}
		if _, seen := toolSet[t]; seen {
			return
		}
		toolSet[t] = struct{}{}
		tools = append(tools, t)
	}
	for _, t := range w.Tools {
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
		for _, t := range am.Tools {
			add(t)
		}
	}

	m := world.Manifest{Tools: tools}

	return &projectWorld{
		Project:    p,
		Name:       name,
		Agents:     append([]string(nil), w.Agents...),
		Workspaces: append([]string(nil), w.Workspaces...),
		Manifest:   m,
	}, nil
}
