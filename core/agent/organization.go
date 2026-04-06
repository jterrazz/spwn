// Package agent provides organization management for defining role structures.
//
// An organization is a first-class entity stored as a YAML file under ~/.spwn/organizations/.
// Each organization defines a set of roles with levels, command relationships,
// and permissions that govern how agents interact within a world.

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"spwn.sh/core/foundation"
)

// Role defines a position within an organization with its level and permissions.
type Role struct {
	Name        string   `json:"name" yaml:"name"`
	Level       int      `json:"level" yaml:"level"`
	CanCommand  []string `json:"can_command,omitempty" yaml:"can_command,omitempty"`
	ReportsTo   string   `json:"reports_to,omitempty" yaml:"reports_to,omitempty"`
	MaxPerWorld int      `json:"max_per_world,omitempty" yaml:"max_per_world,omitempty"`
	Permissions []string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

// Organization is a named set of roles that defines an organisational structure.
type Organization struct {
	Slug        string `json:"slug" yaml:"-"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Roles       []Role `json:"roles" yaml:"roles"`
}

func organizationPath(slug string) string {
	return filepath.Join(foundation.OrganizationsDir(), slug+".yaml")
}

// ValidateOrganization checks that an organization has a valid slug, name, and at
// least one role. It also ensures role names are unique and that can_command
// and reports_to references point to existing roles.
func ValidateOrganization(h Organization) error {
	if h.Slug == "" {
		return fmt.Errorf("organization slug is required")
	}
	if !slugRe.MatchString(h.Slug) {
		return fmt.Errorf("invalid organization slug: %q", h.Slug)
	}
	if h.Name == "" {
		return fmt.Errorf("organization name is required")
	}
	if len(h.Roles) == 0 {
		return fmt.Errorf("organization must have at least one role")
	}

	names := make(map[string]struct{}, len(h.Roles))
	for _, r := range h.Roles {
		if r.Name == "" {
			return fmt.Errorf("role name is required")
		}
		if _, dup := names[r.Name]; dup {
			return fmt.Errorf("duplicate role name: %q", r.Name)
		}
		names[r.Name] = struct{}{}
	}

	// Validate references after collecting all names.
	for _, r := range h.Roles {
		if r.ReportsTo != "" {
			if _, ok := names[r.ReportsTo]; !ok {
				return fmt.Errorf("role %q reports_to unknown role %q", r.Name, r.ReportsTo)
			}
		}
		for _, target := range r.CanCommand {
			if _, ok := names[target]; !ok {
				return fmt.Errorf("role %q can_command unknown role %q", r.Name, target)
			}
		}
	}
	return nil
}

// CreateOrganization persists a new organization. Returns an error if the slug
// already exists or if validation fails.
func CreateOrganization(h Organization) error {
	if h.Slug == "" {
		h.Slug = Slugify(h.Name)
	}
	if err := ValidateOrganization(h); err != nil {
		return err
	}
	dir := foundation.OrganizationsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create organizations dir: %w", err)
	}
	path := organizationPath(h.Slug)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("organization %q already exists", h.Slug)
	}
	return writeOrganization(path, h)
}

// GetOrganization reads an organization by slug.
func GetOrganization(slug string) (*Organization, error) {
	data, err := os.ReadFile(organizationPath(slug))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("organization %q not found", slug)
		}
		return nil, err
	}
	var h Organization
	if err := yaml.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse organization %q: %w", slug, err)
	}
	h.Slug = slug
	return &h, nil
}

// ListOrganizations returns all organizations sorted alphabetically by slug.
func ListOrganizations() ([]Organization, error) {
	dir := foundation.OrganizationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var organizations []Organization
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".yaml")
		h, err := GetOrganization(slug)
		if err != nil {
			continue // skip corrupted files
		}
		organizations = append(organizations, *h)
	}
	return organizations, nil
}

// UpdateOrganization overwrites an existing organization's metadata.
func UpdateOrganization(h Organization) error {
	if h.Slug == "" {
		return fmt.Errorf("organization slug is required")
	}
	if err := ValidateOrganization(h); err != nil {
		return err
	}
	path := organizationPath(h.Slug)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("organization %q not found", h.Slug)
	}
	return writeOrganization(path, h)
}

// DeleteOrganization removes an organization file.
func DeleteOrganization(slug string) error {
	path := organizationPath(slug)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("organization %q not found", slug)
	}
	return os.Remove(path)
}

// GetRole returns a pointer to the named role within the organization, or nil
// if the role does not exist.
func (h *Organization) GetRole(name string) *Role {
	for i := range h.Roles {
		if h.Roles[i].Name == name {
			return &h.Roles[i]
		}
	}
	return nil
}

// RoleCanCommand reports whether the role identified by roleName is allowed
// to command the role identified by targetName.
func (h *Organization) RoleCanCommand(roleName, targetName string) bool {
	r := h.GetRole(roleName)
	if r == nil {
		return false
	}
	for _, t := range r.CanCommand {
		if t == targetName {
			return true
		}
	}
	return false
}

func writeOrganization(path string, h Organization) error {
	data, err := yaml.Marshal(h)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
