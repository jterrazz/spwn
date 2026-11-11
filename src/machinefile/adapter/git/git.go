package git

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: a commit as git names it whole; a branch or a tag moves, a
// source is pinned.
var commit = regexp.MustCompile(`^[0-9a-f]{40}$`)

type Source struct {
	run local.Runner
	fs  local.FS
	dir string
	src contract.SharedSource
}

func New(run local.Runner, fsys local.FS, cache string, src contract.SharedSource) Source {
	// value: the source read from its clone, `<cache>/sources/<name>`.
	return Source{run: run, fs: fsys, dir: filepath.Join(cache, "sources", src.Name), src: src}
}

func (s Source) Read(ctx context.Context) (machinefile.Tree, error) {
	if err := s.checkout(ctx); err != nil {
		return machinefile.Tree{}, &contract.UnavailableError{Source: s.src.Name, Commit: s.src.Commit, Cause: err}
	}
	root := s.src.Name + "@" + s.src.Commit[:7]
	files, err := contract.ReadFolders(s.fs, filepath.Join(s.dir, "catalog"), filepath.Join(s.dir, "machinefiles"))
	return machinefile.Tree{Layer: machinefile.Shared, Root: root, Files: files}, err
}

// invariant: the clone is station's alone and only ever checked out: at
// the pinned commit already, nothing runs; else it is cloned once, fetched
// when the commit is new to it, and checked out detached at the commit.
func (s Source) checkout(ctx context.Context) error {
	if !commit.MatchString(s.src.Commit) {
		return errors.New("its commit " + s.src.Commit + " is no commit: a source is pinned by its 40-character sha")
	}
	if head, err := s.fs.ReadFile(filepath.Join(s.dir, ".git", "HEAD")); err == nil && strings.TrimSpace(string(head)) == s.src.Commit {
		return nil
	}
	if _, err := s.fs.Stat(filepath.Join(s.dir, ".git")); err != nil {
		if err := s.git(ctx, "clone", "--quiet", "--no-checkout", s.src.URL, s.dir); err != nil {
			return err
		}
	} else if err := s.git(ctx, "-C", s.dir, "fetch", "--quiet", "origin", s.src.Commit); err != nil {
		return err
	}
	return s.git(ctx, "-C", s.dir, "-c", "advice.detachedHead=false", "checkout", "--quiet", "--detach", s.src.Commit)
}

func (s Source) git(ctx context.Context, args ...string) error {
	out, err := s.run.Output(ctx, local.Cmd{Name: "git", Args: args})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errors.New(strings.TrimSpace(out.Stderr))
	}
	return nil
}
