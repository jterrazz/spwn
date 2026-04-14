package manifest

import (
	"os"
	"path/filepath"

	intmanifest "spwn.sh/packages/manifest/internal/manifest"
)

// resolveRefs walks the declared agents and world in the manifest and
// turns each name into a filesystem path + existence flag. It does
// NOT parse the referenced files - that's the loader's job. Its only
// purpose is to give callers enough info to produce good error
// messages and to let the validator check structure.
func resolveRefs(root string, m *intmanifest.Manifest) ([]AgentRef, WorldRef) {
	agents := make([]AgentRef, 0, len(m.Agents))
	for _, name := range m.Agents {
		path := filepath.Join(root, "spwn", "agents", name)
		agents = append(agents, AgentRef{
			Name:   name,
			Path:   path,
			Exists: dirExists(path),
		})
	}
	worldPath := filepath.Join(root, "spwn", "worlds", m.World+".yaml")
	world := WorldRef{
		Name:   m.World,
		Path:   worldPath,
		Exists: fileExists(worldPath),
	}
	return agents, world
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
