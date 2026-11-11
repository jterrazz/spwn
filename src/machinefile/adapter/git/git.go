package git

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: a commit as git names it whole; a branch or a tag moves, a
// source is pinned.
var commit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// value: what every git run of a source is given before its verb: no hook
// of the clone, the owner's or a template's runs, and no submodule is
// cloned or checked out, so reading a source runs no code of it.
func quiet() []string {
	return []string{"-c", "core.hooksPath=/dev/null", "-c", "submodule.recurse=false"}
}

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

// invariant: the clone is station's alone and only ever checked out. A
// source is named as a Machinefile is, so its clone is one folder of the
// cache; a clone already there is trusted only untouched (no change, no file
// git does not track) and is read with nothing fetched when its HEAD is
// the pinned commit; else it is cloned once, fetched from its url when
// the commit is new to it, and checked out detached at the commit.
func (s Source) checkout(ctx context.Context) error {
	switch {
	case !catalog.IsMachinefileName(s.src.Name):
		return errors.New("its name " + strconv.Quote(s.src.Name) + " is no name: lower-case words joined by dashes (`studio`)")
	case !commit.MatchString(s.src.Commit):
		return errors.New("its commit " + s.src.Commit + " is no commit: a source is pinned by its 40-character sha")
	}
	if _, err := s.fs.Stat(filepath.Join(s.dir, ".git")); err != nil {
		if _, err := s.git(ctx, "", "clone", "--quiet", "--no-checkout", "--", s.src.URL, s.dir); err != nil {
			return err
		}
		return s.detach(ctx)
	}
	head, untouched, err := s.status(ctx)
	switch {
	case err != nil:
		return err
	case !untouched:
		return errors.New("its clone " + s.dir + " holds changes: station only checks it out, so delete it and station clones it again")
	case head == s.src.Commit:
		return nil
	}
	if _, err := s.git(ctx, s.dir, "fetch", "--quiet", "--no-recurse-submodules", "--", s.src.URL, s.src.Commit); err != nil {
		return err
	}
	return s.detach(ctx)
}

func (s Source) detach(ctx context.Context) error {
	_, err := s.git(ctx, s.dir, "-c", "advice.detachedHead=false", "checkout", "--quiet", "--detach", s.src.Commit, "--")
	return err
}

// value: the commit a clone's HEAD is at, and whether its working tree is
// untouched: nothing changed, nothing untracked, nothing ignored.
func (s Source) status(ctx context.Context) (head string, untouched bool, err error) {
	out, err := s.git(ctx, s.dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")
	if err != nil {
		return "", false, err
	}
	untouched = true
	for line := range strings.Lines(out) {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
		case strings.HasPrefix(line, "# branch.oid "):
			head = strings.TrimPrefix(line, "# branch.oid ")
		case strings.HasPrefix(line, "#"):
		default:
			untouched = false
		}
	}
	return head, untouched, nil
}

// value: git run on its own, or in a clone (`-C <dir>`), quiet first.
func (s Source) git(ctx context.Context, in string, args ...string) (string, error) {
	var argv []string
	if in != "" {
		argv = append(argv, "-C", in)
	}
	argv = append(append(argv, quiet()...), args...)
	out, err := s.run.Output(ctx, local.Cmd{Name: "git", Args: argv})
	if err != nil {
		return "", err
	}
	if out.ExitCode != 0 {
		return "", errors.New(strings.TrimSpace(out.Stderr))
	}
	return out.Stdout, nil
}
