package project

import (
	"errors"
	"io/fs"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"spwn.sh/packages/platform"
)

// TeamMembers returns the names of all agents whose agent.yaml
// references the given team slug.
func TeamMembers(teamSlug string) ([]string, error) {
	files := platform.DataFiles()
	defer files.Close()
	agentsDir := platform.AgentsDir()
	entries, err := files.ReadDir(agentsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var members []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := files.ReadFile(filepath.Join(agentsDir, e.Name(), "agent.yaml"))
		if err != nil {
			continue
		}
		var p struct {
			Team string `yaml:"team"`
		}
		if yaml.Unmarshal(data, &p) == nil && p.Team == teamSlug {
			members = append(members, e.Name())
		}
	}
	return members, nil
}
