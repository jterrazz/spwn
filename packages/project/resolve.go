package project

import (
	"os"
	"path/filepath"
	"sort"

	intmanifest "spwn.sh/packages/project/internal/manifest"
)

// resolveRefs walks the agents declared by every world in the manifest
// and turns each name into a filesystem path + existence flag. It also
// scans spwn/agents/ on disk for "orphan" directories — agents that
// exist but aren't referenced by any world.
func resolveRefs(root string, m *intmanifest.Manifest) (deployable, orphans []AgentRef) {
	declared := map[string]struct{}{}
	for _, w := range m.Worlds {
		for _, name := range w.Agents {
			declared[name] = struct{}{}
		}
	}

	// Deployable agents: the union of names from every world.
	names := make([]string, 0, len(declared))
	for n := range declared {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(root, "spwn", "agents", name)
		deployable = append(deployable, AgentRef{
			Name:   name,
			Path:   path,
			Exists: dirExists(path),
		})
	}

	// Orphans: directories under spwn/agents/ not in declared.
	agentsDir := filepath.Join(root, "spwn", "agents")
	entries, err := os.ReadDir(agentsDir)
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

func dirExists(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return info.IsDir()
}
