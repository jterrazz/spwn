package files

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"spwn.sh/packages/platform"
	spwnproject "spwn.sh/packages/project"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	projectcontract "github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/skills"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
)

type disk interface {
	MkdirAll(path string, perm fs.FileMode) error
}

type Files struct {
	// value: the disk the folder that holds the user's own skills is made
	// on, trusted as the caller's.
	fs disk
	// value: the skills' folder and each file in it, reached from the
	// project's root one name at a time, never through a link.
	under local.Beneath
	// value: the layout of the project a skill is read and written in.
	layouts projectcontract.Layouts
}

var _ contract.Files = Files{}

func New(fsys disk, layouts projectcontract.Layouts) Files { return Files{fs: fsys, layouts: layouts} }

// invariant: Agents' packages resolve every path from one process-global
// project, which the Layouts hold for one act at the folders its spwn.yaml
// lays out, so the folder is read inside it. A project's folder is reached
// from its root, the user's own from the folder that holds it, each the
// caller's to trust.
func in[T any](l projectcontract.Layouts, project string, act func(base, rel, dir string) (T, error)) (T, error) {
	var out T
	err := l.Within(project, func() error {
		var err error
		dir := platform.SkillsDir()
		if project != "" {
			if rel, relErr := filepath.Rel(project, dir); relErr == nil && filepath.IsLocal(rel) {
				out, err = act(project, filepath.ToSlash(rel), dir)
				return err
			}
		}
		out, err = act(filepath.Dir(dir), filepath.Base(dir), dir)
		return err
	})
	return out, err
}

func fileOf(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return "", skills.ErrNotFound
	}
	return name + skills.Suffix, nil
}

func (f Files) List(_ context.Context, project string) ([]skills.Skill, error) {
	return in(f.layouts, project, func(base, rel, dir string) ([]skills.Skill, error) {
		folder, err := f.under.Open(base, rel)
		if errors.Is(err, fs.ErrNotExist) {
			return []skills.Skill{}, nil
		}
		if err != nil {
			return nil, err
		}
		defer folder.Close()
		names, err := folder.Names()
		if err != nil {
			return nil, err
		}
		out := []skills.Skill{}
		for _, file := range names {
			name, ok := strings.CutSuffix(file, skills.Suffix)
			if !ok {
				continue
			}
			if e, err := folder.Lstat(file); err != nil || !e.Mode.IsRegular() {
				continue
			}
			body, _, err := folder.ReadFile(file)
			if errors.Is(err, local.ErrTooLarge) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if !spwnproject.SkillFile(file, body) {
				continue
			}
			out = append(out, skills.Skill{Name: name, Path: filepath.Join(dir, file), Description: skills.Describe(string(body))})
		}
		slices.SortFunc(out, func(a, b skills.Skill) int { return strings.Compare(a.Name, b.Name) })
		return out, nil
	})
}

func (f Files) Show(_ context.Context, project, name string) (skills.Skill, error) {
	return in(f.layouts, project, func(base, rel, dir string) (skills.Skill, error) {
		file, err := fileOf(name)
		if err != nil {
			return skills.Skill{}, err
		}
		folder, err := f.under.Open(base, rel)
		if errors.Is(err, fs.ErrNotExist) {
			return skills.Skill{}, skills.ErrNotFound
		}
		if err != nil {
			return skills.Skill{}, err
		}
		defer folder.Close()
		body, _, err := folder.ReadFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			return skills.Skill{}, skills.ErrNotFound
		}
		if err != nil {
			return skills.Skill{}, err
		}
		return skills.Skill{Name: name, Path: filepath.Join(dir, file), Description: skills.Describe(string(body)), Body: string(body)}, nil
	})
}

func (f Files) Create(_ context.Context, project, name string) (skills.Skill, error) {
	return in(f.layouts, project, func(base, rel, dir string) (skills.Skill, error) {
		file, unnamed := fileOf(name)
		if unnamed != nil {
			return skills.Skill{}, skills.ErrBadName
		}
		if project == "" {
			if err := f.fs.MkdirAll(base, 0o755); err != nil {
				return skills.Skill{}, err
			}
		}
		if err := f.under.Folders(base, rel, 0o755, local.Self()); err != nil {
			return skills.Skill{}, err
		}
		folder, err := f.under.Open(base, rel)
		if err != nil {
			return skills.Skill{}, err
		}
		defer folder.Close()
		if _, err = folder.Lstat(file); err == nil {
			return skills.Skill{}, skills.ErrExists
		}
		made := skills.Blank(name, filepath.Join(dir, file))
		err = folder.Replace(file, local.Seen{}, []byte(made.Body), 0o644, local.Self())
		if errors.Is(err, local.ErrChanged) {
			return skills.Skill{}, skills.ErrExists
		}
		return made, err
	})
}

func (f Files) Delete(_ context.Context, project, name string) error {
	_, err := in(f.layouts, project, func(base, rel, _ string) (struct{}, error) {
		file, err := fileOf(name)
		if err != nil {
			return struct{}{}, err
		}
		folder, err := f.under.Open(base, rel)
		if errors.Is(err, fs.ErrNotExist) {
			return struct{}{}, skills.ErrNotFound
		}
		if err != nil {
			return struct{}{}, err
		}
		defer folder.Close()
		body, _, err := folder.ReadFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			return struct{}{}, skills.ErrNotFound
		}
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, folder.Remove(file, local.Seen{Data: body, There: true})
	})
	return err
}
