package source

import (
	"fmt"
	"sort"

	"spwn.sh/packages/compile"
	"spwn.sh/packages/world/models"
)

// ToCompileInput projects a ProjectSource onto the compile.Input shape
// that compile.Runtime.Render expects. worldName selects which world
// from the manifest to compile; pass an empty string and ToCompileInput
// will pick "the only world" (error if multiple worlds exist).
//
// Tool verification is skipped here — callers feeding the result
// through a real spawn can override VerifiedTools; for CLI dry-runs
// the union of agent tools is the best we can do without probing a
// container.
func ToCompileInput(src *ProjectSource, worldName string) (compile.Input, error) {
	if src == nil {
		return compile.Input{}, fmt.Errorf("nil ProjectSource")
	}
	if src.Manifest == nil {
		return compile.Input{}, fmt.Errorf("ProjectSource has no manifest")
	}

	worlds := src.Manifest.Worlds
	if len(worlds) == 0 {
		return compile.Input{}, fmt.Errorf("no worlds declared in spwn.yaml")
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
			return compile.Input{}, fmt.Errorf(
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
		return compile.Input{}, fmt.Errorf("world %q not in manifest (have: %v)", selected, names)
	}

	// Index agents on disk by name.
	byName := make(map[string]AgentSource, len(src.Agents))
	for _, a := range src.Agents {
		byName[a.Name] = a
	}

	// Collect the union of packages from every agent in this world.
	// This mirrors what spawn does before probing the container: the
	// render doesn't need a verified list, it just needs to know what
	// the manifest *claims* is available.
	packages := map[string]struct{}{}
	agents := make([]compile.AgentInput, 0, len(world.Agents))
	for _, name := range world.Agents {
		a, ok := byName[name]
		if !ok {
			return compile.Input{}, fmt.Errorf(
				"world %q references missing agent %q", selected, name)
		}
		for _, p := range a.Config.Deps {
			packages[p] = struct{}{}
		}
		agents = append(agents, compile.AgentInput{
			Name: a.Name,
			Role: a.Config.Role,
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

	return compile.Input{
		Manifest: models.Manifest{
			Deps: packageList,
		},
		VerifiedTools: packageList,
		WorldID:       selected,
		Agents:        agents,
	}, nil
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
