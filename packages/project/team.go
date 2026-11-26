package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"spwn.sh/packages/platform"
)

// Team is a named group of agents with display metadata.
type Team struct {
	Slug        string `json:"slug" yaml:"-"`
	Name        string `json:"name" yaml:"name"`
	Color       string `json:"color,omitempty" yaml:"color,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// Slugify converts a display name to a filesystem-safe slug.
// "Matrix Ops" gives "matrix-ops", "infra" gives "infra".
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
	return filepath.Join(platform.TeamsDir(), slug+".yaml")
}

// CreateTeam persists a new team. Returns an error if the slug already exists.
func CreateTeam(t Team) error {
	if t.Slug == "" {
		t.Slug = Slugify(t.Name)
	}
	if !slugRe.MatchString(t.Slug) {
		return fmt.Errorf("invalid team slug: %q", t.Slug)
	}
	dir := platform.TeamsDir()
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
	dir := platform.TeamsDir()
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
			continue
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

func writeTeam(path string, t Team) error {
	data, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
