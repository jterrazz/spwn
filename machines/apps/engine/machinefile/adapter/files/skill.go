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
	Readlink(path string) (string, error)
	Replace(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// value: the machine's `spwn` skill, one SKILL.md per agent skill folder
// of the user (`~/.agents/skills/spwn`, `~/.claude/skills/spwn`), owned
// whole by Machines: the `spwn` folder and its file are Machines' own.

type Skill struct {
	fs    replacing
	paths []string
}

var _ contract.Skill = Skill{}

func NewSkill(fsys replacing, paths ...string) Skill { return Skill{fs: fsys, paths: paths} }

// value: a skill folder or file that is a link: Machines writes neither
// through it nor over it.
var errLink = errors.New("is a link")

// invariant: each copy is judged on its own, every one tried: a `spwn`
// folder or a SKILL.md that is a link is refused (errLink) and left as it
// is, never followed nor replaced; a copy whose bytes are already the body
// is left untouched; every other copy, a file written by hand there
// included, is replaced whole and durably (FS.Replace). The first copy
// refused or failed is the error, naming its own path; the folders above
// `spwn` are the user's, followed as they stand, so two skill folders that
// are one through a link are written once.

func (s Skill) Write(_ context.Context, body []byte) (bool, error) {
	changed := false
	var failed error
	for _, at := range s.paths {
		wrote, err := s.write(at, body)
		changed = changed || wrote
		if err != nil && failed == nil {
			failed = err
		}
	}
	return changed, failed
}

func (s Skill) write(at string, body []byte) (bool, error) {
	for _, own := range []string{filepath.Dir(at), at} {
		if _, err := s.fs.Readlink(own); err == nil {
			return false, &fs.PathError{Op: "write", Path: own, Err: errLink}
		}
	}
	switch there, err := s.fs.ReadFile(at); {
	case err == nil && bytes.Equal(there, body):
		return false, nil
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	if err := s.fs.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return false, err
	}
	if err := s.fs.Replace(at, body, 0o644); err != nil {
		return false, err
	}
	return true, nil
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
