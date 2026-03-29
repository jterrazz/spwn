package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// SkillInfo describes an installed skill.
type SkillInfo struct {
	Name   string
	Source string
}

// LocalRegistry implements the Skill port using the local filesystem.
type LocalRegistry struct {
	dir string // ~/.spwn/skills/
}

// NewLocal creates a new LocalRegistry adapter.
func NewLocal(dir string) *LocalRegistry {
	return &LocalRegistry{dir: dir}
}

// List returns all locally installed skills.
func (r *LocalRegistry) List(ctx context.Context) ([]SkillInfo, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var skills []SkillInfo
	for _, e := range entries {
		if e.IsDir() {
			skills = append(skills, SkillInfo{Name: e.Name(), Source: "local"})
		}
	}
	return skills, nil
}

// Install copies a skill from the given source into the local registry.
func (r *LocalRegistry) Install(ctx context.Context, source string) error {
	return fmt.Errorf("not yet implemented")
}

// Remove deletes a skill from the local registry.
func (r *LocalRegistry) Remove(ctx context.Context, name string) error {
	return os.RemoveAll(filepath.Join(r.dir, name))
}
