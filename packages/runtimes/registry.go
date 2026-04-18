package runtimes

import "fmt"

var spawners = map[string]Spawner{}

// RegisterSpawner adds a runtime's spawn-time adapter to the global
// spawner registry. Typically called from each runtime subpackage's
// init() so the CLI can resolve a spawner by name via GetSpawner.
func RegisterSpawner(s Spawner) {
	spawners[s.Name()] = s
}

// GetSpawner returns the spawn-time adapter for a runtime.
func GetSpawner(name string) (Spawner, error) {
	s, ok := spawners[name]
	if !ok {
		names := make([]string, 0, len(spawners))
		for k := range spawners {
			names = append(names, k)
		}
		return nil, fmt.Errorf("unknown runtime %q, available: %v", name, names)
	}
	return s, nil
}

// AllSpawners returns every registered spawn-time adapter keyed by
// runtime name. Callers must not mutate the returned map.
func AllSpawners() map[string]Spawner {
	return spawners
}
