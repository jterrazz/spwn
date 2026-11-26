package project

import (
	"errors"
	"fmt"
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

// SetAgentTeam updates the agent's agent.yaml to reference the given
// team slug. An empty slug clears the team assignment. In a project, an
// agent.yaml that is a link, or under one, is refused, never written
// through.
func SetAgentTeam(agentName, teamSlug string) error {
	if err := platform.CheckName("agent", agentName); err != nil {
		return err
	}
	files := platform.DataFiles()
	defer files.Close()
	agentDir := filepath.Join(platform.AgentsDir(), agentName)
	if _, err := files.Stat(agentDir); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("agent %q not found", agentName)
	}
	manifestPath := filepath.Join(agentDir, "agent.yaml")

	var m map[string]any
	data, err := files.ReadFile(manifestPath)
	switch {
	case err == nil:
		_ = yaml.Unmarshal(data, &m)
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("read the agent.yaml of %q: %w", agentName, err)
	}
	if m == nil {
		m = map[string]any{}
	}

	if teamSlug == "" {
		delete(m, "team")
	} else {
		m["team"] = teamSlug
	}

	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return files.WriteFile(manifestPath, out, 0o644)
}
