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

// value: a commit as git names it whole, the one a clone's HEAD must be
// at to be read.
var commit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// value: what every git run of a source is given before its verb: no hook
// of the clone, the owner's or a template's runs, and no submodule is
// cloned or checked out, so reading a source runs no code of it; and no
// transport but https and ssh is spoken (`ext::` runs a command, `file`
// reads any folder of the machine).
func quiet() []string {
	return []string{
		"-c", "core.hooksPath=/dev/null", "-c", "submodule.recurse=false",
		"-c", "protocol.allow=never", "-c", "protocol.https.allow=always", "-c", "protocol.ssh.allow=always",
	}
}

// value: a source's url as git may be given it: `https://`, `ssh://` or
// scp-style `user@domain:path`; the first character is never a dash, and
// `file://`, `ext::`, a path or any other scheme is no url of a source.
var remote = regexp.MustCompile(`^(?:https://[^\s/]|ssh://[^\s/]|[A-Za-z0-9._-]+@[A-Za-z0-9.-]+:[^\s:])`)

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
	head, _, err := s.clone(ctx)
	if err != nil {
		return machinefile.Tree{}, &contract.UnavailableError{Source: s.src.Name, Cause: err}
	}
	files, err := contract.ReadFolders(s.fs, contract.NoLinks, filepath.Join(s.dir, "catalog"), filepath.Join(s.dir, "machinefiles"))
	return machinefile.Tree{
		Layer: machinefile.Shared, Root: s.src.Name + "@" + head[:7], Files: files,
		Source: &catalog.LockSource{Name: s.src.Name, Commit: head},
	}, err
}

func (s Source) Pull(ctx context.Context) error {
	// invariant: the commit the source's default branch is at now is
	// fetched from its url and checked out detached; a clone made by this
	// very call is already there and fetches nothing more.
	_, fresh, err := s.clone(ctx)
	if err == nil && !fresh {
		if _, err = s.git(ctx, s.dir, "fetch", "--quiet", "--no-recurse-submodules", "--", s.src.URL, "HEAD"); err == nil {
			_, err = s.git(ctx, s.dir, "-c", "advice.detachedHead=false", "checkout", "--quiet", "--detach", "FETCH_HEAD", "--")
		}
	}
	if err != nil {
		return &contract.UnavailableError{Source: s.src.Name, Cause: err}
	}
	return nil
}

// invariant: the clone is station's alone and only ever checked out. A
// source is named as a Machinefile is, so its clone is one folder of the
// cache, cloned once at the source's default branch when it is not
// there; a clone already there is trusted only untouched (no change, no
// file git does not track) and is read where it stands, with nothing
// fetched. It answers the commit the clone's HEAD is at, and whether this
// call cloned it.
func (s Source) clone(ctx context.Context) (head string, fresh bool, err error) {
	switch {
	case !catalog.IsMachinefileName(s.src.Name):
		return "", false, errors.New("its name " + strconv.Quote(s.src.Name) + " is no name: lower-case words joined by dashes (`studio`)")
	case !remote.MatchString(s.src.URL):
		return "", false, errors.New("its url " + strconv.Quote(s.src.URL) + " is no url station clones: https://, ssh:// or user@domain:path")
	}
	if _, missing := s.fs.Stat(filepath.Join(s.dir, ".git")); missing != nil {
		if _, err = s.git(ctx, "", "clone", "--quiet", "--", s.src.URL, s.dir); err != nil {
			return "", false, err
		}
		fresh = true
	}
	head, untouched, err := s.status(ctx)
	switch {
	case err != nil:
		return "", false, err
	case !untouched:
		return "", false, errors.New("its clone " + s.dir + " holds changes: station only checks it out, so delete it and station clones it again")
	case !commit.MatchString(head):
		return "", false, errors.New("its clone " + s.dir + " is at no commit: its default branch holds none")
	}
	return head, fresh, nil
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
