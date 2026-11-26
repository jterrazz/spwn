package project

import (
	"path/filepath"
	"sort"

	"spwn.sh/packages/platform"
	intmanifest "spwn.sh/packages/project/internal/manifest"
	"spwn.sh/packages/project/internal/validate"
)

// invariant: a declared name that is not one folder name (a separator,
// "..", a NUL) names no folder: it is never joined to a path, and
// Validate says why.
func resolveRefs(t *platform.Tree, root string, m *intmanifest.Manifest) (deployable, orphans []AgentRef) {
	declared := map[string]struct{}{}
	for _, w := range m.Worlds {
		for _, name := range w.Agents {
			declared[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(declared))
	for n := range declared {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if !validate.IsOneName(name) {
			deployable = append(deployable, AgentRef{Name: name})
			continue
		}
		path := filepath.Join(root, "spwn", "agents", name)
		deployable = append(deployable, AgentRef{
			Name:   name,
			Path:   path,
			Exists: t.IsDir(path),
		})
	}

	agentsDir := filepath.Join(root, "spwn", "agents")
	entries, err := t.ReadDir(agentsDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if _, ok := declared[name]; ok {
				continue
			}
			path := filepath.Join(agentsDir, name)
			orphans = append(orphans, AgentRef{
				Name:   name,
				Path:   path,
				Exists: true,
			})
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].Name < orphans[j].Name })
	return deployable, orphans
}
