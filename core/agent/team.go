// Package agent provides team management for grouping agents.
//
// A team is a first-class entity stored as a YAML file under ~/.spwn/teams/.
// Each agent's profile can reference a team by slug. Teams carry display
// metadata (name, icon, color, description) for the UI.

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"spwn.sh/core/foundation"
	"gopkg.in/yaml.v3"
)

// Team is a named group of agents with display metadata.
type Team struct {
	Slug        string `json:"slug" yaml:"-"`
	Name        string `json:"name" yaml:"name"`
	Icon        string `json:"icon,omitempty" yaml:"icon,omitempty"`
	Color       string `json:"color,omitempty" yaml:"color,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// Slugify converts a display name to a filesystem-safe slug.
// "Matrix Ops" → "matrix-ops", "infra" → "infra".
func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "team"
	}
	return s
}

func teamPath(slug string) string {
	return filepath.Join(foundation.TeamsDir(), slug+".yaml")
}

// CreateTeam persists a new team. Returns an error if the slug already exists.
func CreateTeam(t Team) error {
	if t.Slug == "" {
		t.Slug = Slugify(t.Name)
	}
	if !slugRe.MatchString(t.Slug) {
		return fmt.Errorf("invalid team slug: %q", t.Slug)
	}
	dir := foundation.TeamsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create teams dir: %w", err)
	}
	path := teamPath(t.Slug)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("team %q already exists", t.Slug)
	}
	return writeTeam(path, t)
}

// GetTeam reads a team by slug.
func GetTeam(slug string) (*Team, error) {
	data, err := os.ReadFile(teamPath(slug))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("team %q not found", slug)
		}
		return nil, err
	}
	var t Team
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse team %q: %w", slug, err)
	}
	t.Slug = slug
	return &t, nil
}

// ListTeams returns all teams sorted alphabetically by slug.
func ListTeams() ([]Team, error) {
	dir := foundation.TeamsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var teams []Team
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".yaml")
		t, err := GetTeam(slug)
		if err != nil {
			continue // skip corrupted files
		}
		teams = append(teams, *t)
	}
	return teams, nil
}

// UpdateTeam overwrites an existing team's metadata.
func UpdateTeam(t Team) error {
	if t.Slug == "" {
		return fmt.Errorf("team slug is required")
	}
	path := teamPath(t.Slug)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("team %q not found", t.Slug)
	}
	return writeTeam(path, t)
}

// DeleteTeam removes a team file. Agents referencing it become solo
// (their profile.team still holds the slug but the team entity is gone).
func DeleteTeam(slug string) error {
	path := teamPath(slug)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("team %q not found", slug)
	}
	return os.Remove(path)
}

// TeamMembers returns the names of all agents whose profile.yaml
// references the given team slug.
func TeamMembers(teamSlug string) ([]string, error) {
	agentsDir := foundation.AgentsDir()
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var members []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		profilePath := filepath.Join(agentsDir, e.Name(), "profile.yaml")
		data, err := os.ReadFile(profilePath)
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

// SetAgentTeam updates the agent's profile.yaml to reference the given
// team slug. An empty slug clears the team assignment.
func SetAgentTeam(agentName, teamSlug string) error {
	agentDir := filepath.Join(foundation.AgentsDir(), agentName)
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		return fmt.Errorf("agent %q not found", agentName)
	}
	profilePath := filepath.Join(agentDir, "profile.yaml")

	// Read existing profile (if any).
	var profile map[string]any
	data, err := os.ReadFile(profilePath)
	if err == nil {
		_ = yaml.Unmarshal(data, &profile)
	}
	if profile == nil {
		profile = map[string]any{}
	}

	if teamSlug == "" {
		delete(profile, "team")
	} else {
		profile["team"] = teamSlug
	}

	out, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}
	return os.WriteFile(profilePath, out, 0644)
}

func writeTeam(path string, t Team) error {
	data, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
