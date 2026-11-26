package files

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"spwn.sh/packages/platform"

	"github.com/jterrazz/studio-station/apps/engine/skills"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
)

type disk interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	Stat(path string) (fs.FileInfo, error)
}

type Files struct {
	// value: the disk of the machine the skills are kept on.
	fs disk
}

var _ contract.Files = Files{}

func New(fsys disk) Files { return Files{fs: fsys} }

// invariant: Agents' packages resolve every path from one process-global
// project root, which platform.WithProjectRoot holds for one act, so the
// folder is read inside it.
func in[T any](project string, act func(dir string) (T, error)) (T, error) {
	return platform.WithProjectRoot(project, func() (T, error) { return act(platform.SkillsDir()) })
}

func fileOf(dir, name string) (string, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return "", skills.ErrNotFound
	}
	return filepath.Join(dir, name+skills.Suffix), nil
}

func (f Files) List(_ context.Context, project string) ([]skills.Skill, error) {
	return in(project, func(dir string) ([]skills.Skill, error) {
		entries, err := f.fs.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			return []skills.Skill{}, nil
		}
		if err != nil {
			return nil, err
		}
		out := []skills.Skill{}
		for _, e := range entries {
			name, ok := strings.CutSuffix(e.Name(), skills.Suffix)
			if !ok || e.IsDir() {
				continue
			}
			at := filepath.Join(dir, e.Name())
			body, err := f.fs.ReadFile(at)
			if err != nil {
				return nil, err
			}
			out = append(out, skills.Skill{Name: name, Path: at, Description: skills.Describe(string(body))})
		}
		slices.SortFunc(out, func(a, b skills.Skill) int { return strings.Compare(a.Name, b.Name) })
		return out, nil
	})
}

func (f Files) Show(_ context.Context, project, name string) (skills.Skill, error) {
	return in(project, func(dir string) (skills.Skill, error) {
		at, err := fileOf(dir, name)
		if err != nil {
			return skills.Skill{}, err
		}
		body, err := f.fs.ReadFile(at)
		if errors.Is(err, fs.ErrNotExist) {
			return skills.Skill{}, skills.ErrNotFound
		}
		if err != nil {
			return skills.Skill{}, err
		}
		return skills.Skill{Name: name, Path: at, Description: skills.Describe(string(body)), Body: string(body)}, nil
	})
}

func (f Files) Create(_ context.Context, project, name string) (skills.Skill, error) {
	return in(project, func(dir string) (skills.Skill, error) {
		at, err := fileOf(dir, name)
		if err != nil {
			return skills.Skill{}, skills.ErrBadName
		}
		if _, err := f.fs.Stat(at); err == nil {
			return skills.Skill{}, skills.ErrExists
		}
		if err := f.fs.MkdirAll(dir, 0o755); err != nil {
			return skills.Skill{}, err
		}
		made := skills.Blank(name, at)
		return made, f.fs.WriteFile(at, []byte(made.Body), 0o644)
	})
}

func (f Files) Delete(_ context.Context, project, name string) error {
	_, err := in(project, func(dir string) (struct{}, error) {
		at, err := fileOf(dir, name)
		if err != nil {
			return struct{}{}, err
		}
		if err := f.fs.Remove(at); errors.Is(err, fs.ErrNotExist) {
			return struct{}{}, skills.ErrNotFound
		} else if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}
