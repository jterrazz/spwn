package files

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

type replacing interface {
	ReadFile(path string) ([]byte, error)
	Replace(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// value: the machine's `spwn` skill, one SKILL.md per agent skill folder
// of the user (`~/.agents/skills/spwn`, `~/.claude/skills/spwn`), owned
// whole by Machines.

type Skill struct {
	fs    replacing
	paths []string
}

var _ contract.Skill = Skill{}

func NewSkill(fsys replacing, paths ...string) Skill { return Skill{fs: fsys, paths: paths} }

// invariant: a copy whose bytes are already the body is left untouched, so
// two folders that are one through a link are written once; every other
// copy is replaced whole and durably (FS.Replace).

func (s Skill) Write(_ context.Context, body []byte) (bool, error) {
	changed := false
	for _, at := range s.paths {
		switch there, err := s.fs.ReadFile(at); {
		case err == nil && bytes.Equal(there, body):
			continue
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return changed, err
		}
		if err := s.fs.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			return changed, err
		}
		if err := s.fs.Replace(at, body, 0o644); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

func (s Skill) Read(context.Context) (body []byte, ok bool, err error) {
	if len(s.paths) == 0 {
		return nil, false, nil
	}
	body, err = s.fs.ReadFile(s.paths[0])
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return body, err == nil, err
}
