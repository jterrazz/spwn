package git_test

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/git"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

const (
	pinned    = "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d"
	moved     = "ffffffffffffffffffffffffffffffffffffffff"
	rewritten = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	url       = "https://example.com/studio.git"
)

// value: what every git run of a source is given before its verb, then
// its words.
func quiet(args ...string) []string {
	return append([]string{
		"-c", "core.hooksPath=/dev/null", "-c", "submodule.recurse=false",
		"-c", "protocol.allow=never", "-c", "protocol.https.allow=always", "-c", "protocol.ssh.allow=always",
	}, args...)
}

func in(dir string, args ...string) []string {
	return append([]string{"-C", dir}, quiet(args...)...)
}

// value: a clone as git leaves it: the files at its commit.
func clone(t *testing.T, dir string, held map[string]string) {
	t.Helper()
	held = maps.Clone(held)
	if held == nil {
		held = map[string]string{}
	}
	held[".git/HEAD"] = pinned + "\n"
	for p, body := range held {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// value: git status of a clone at a commit, and the lines of what it
// holds beside it.
func status(at string, held ...string) local.Output {
	lines := append([]string{"# branch.oid " + at, "# branch.head (detached)"}, held...)
	return local.Output{Stdout: strings.Join(lines, "\n") + "\n"}
}

func source(run local.Runner, cache, name string) git.Source {
	return git.New(run, local.New(nil, nil, nil, nil).FS, cache, contract.SharedSource{Name: name, URL: url})
}

func TestTheSharedSourceKeepsTheContract(t *testing.T) {
	contracttest.Source(t, func(t *testing.T, held map[string]string) contract.Source {
		cache := t.TempDir()
		dir := filepath.Join(cache, "sources", "studio")
		run := localtest.NewRunner(t)
		run.On("git", quiet("clone", "--quiet", "--", url, dir)...).Answer(func() local.Output {
			clone(t, dir, held)
			return local.Output{}
		})
		run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Reply(status(pinned))
		return source(run, cache, "studio")
	})
}

func TestACloneIsReadWhereItStandsAndFetchesNothing(t *testing.T) {
	cache := t.TempDir()
	dir := filepath.Join(cache, "sources", "studio")
	clone(t, dir, nil)
	run := localtest.NewRunner(t)
	run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Reply(status(pinned))
	tree, err := source(run, cache, "studio").Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if tree.Root != "studio@1a2b3c4" || tree.Source == nil || tree.Source.Commit != pinned {
		t.Errorf("read as %s from %+v, want studio@1a2b3c4 at %s", tree.Root, tree.Source, pinned)
	}
}

func TestTheSharedSourceFollowsItsBranch(t *testing.T) {
	contracttest.Follows(t, func(t *testing.T) (contract.Source, func() string, func() string) {
		cache := t.TempDir()
		dir := filepath.Join(cache, "sources", "studio")
		clone(t, dir, nil)
		at, branch, fetched, forward := pinned, pinned, "", true
		run := localtest.NewRunner(t)
		run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Answer(func() local.Output { return status(at) })
		run.On("git", in(dir, "fetch", "--quiet", "--no-recurse-submodules", "--", url, "HEAD")...).Answer(func() local.Output {
			fetched = branch
			return local.Output{}
		})
		run.On("git", in(dir, "rev-parse", "--verify", "--quiet", "FETCH_HEAD^{commit}")...).Answer(func() local.Output { return local.Output{Stdout: fetched + "\n"} })
		for _, to := range []string{moved, rewritten} {
			run.On("git", in(dir, "merge-base", "--is-ancestor", pinned, to)...).Answer(func() local.Output {
				if forward {
					return local.Output{}
				}
				return local.Output{ExitCode: 1}
			})
		}
		for _, to := range []string{pinned, moved} {
			run.On("git", in(dir, "-c", "advice.detachedHead=false", "checkout", "--quiet", "--detach", to, "--")...).Answer(func() local.Output {
				at = to
				return local.Output{}
			})
		}
		move := func(commit string, ahead bool) func() string {
			return func() string {
				branch, forward = commit, ahead
				return commit
			}
		}
		return source(run, cache, "studio"), move(moved, true), move(rewritten, false)
	})
}

func TestPullOnNoCloneClonesItOnce(t *testing.T) {
	cache := t.TempDir()
	dir := filepath.Join(cache, "sources", "studio")
	run := localtest.NewRunner(t)
	clones := run.On("git", quiet("clone", "--quiet", "--", url, dir)...).Answer(func() local.Output {
		clone(t, dir, nil)
		return local.Output{}
	})
	run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Reply(status(pinned))
	if _, err := source(run, cache, "studio").Pull(t.Context()); err != nil {
		t.Fatal(err)
	}
	if clones.Seen != 1 {
		t.Errorf("cloned %d times, want once", clones.Seen)
	}
}

func TestACloneThatHoldsChangesIsNotTrusted(t *testing.T) {
	for _, held := range []string{"1 .M N... 100644 100644 100644 a b machinefiles/laptop.yaml", "? machinefiles/planted.yaml", "! catalog/apps/hidden.yaml"} {
		cache := t.TempDir()
		dir := filepath.Join(cache, "sources", "studio")
		clone(t, dir, nil)
		run := localtest.NewRunner(t)
		run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Reply(status(pinned, held))
		_, err := source(run, cache, "studio").Read(t.Context())
		if err == nil || !strings.Contains(err.Error(), "holds changes") {
			t.Errorf("a clone holding %q read with %v", held, err)
		}
		if _, err := source(run, cache, "studio").Pull(t.Context()); err == nil || !strings.Contains(err.Error(), "holds changes") {
			t.Errorf("a clone holding %q pulled with %v", held, err)
		}
	}
}

func TestACloneAtNoCommitIsNotRead(t *testing.T) {
	cache := t.TempDir()
	dir := filepath.Join(cache, "sources", "studio")
	clone(t, dir, nil)
	run := localtest.NewRunner(t)
	run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Reply(status("(initial)"))
	if _, err := source(run, cache, "studio").Read(t.Context()); err == nil || !strings.Contains(err.Error(), "is at no commit") {
		t.Errorf("a clone at no commit read with %v", err)
	}
}

func TestANameThatIsNoNameRunsNothing(t *testing.T) {
	for _, name := range []string{"..", "../escape", "a/b", "-x", "--upload-pack=touch", "Studio", ""} {
		_, err := source(localtest.NewRunner(t), t.TempDir(), name).Read(t.Context())
		if err == nil || !strings.Contains(err.Error(), "is no name") {
			t.Errorf("a source named %q read with %v", name, err)
		}
	}
}

func TestAUrlThatIsNoRemoteRunsNothing(t *testing.T) {
	for _, bad := range []string{"ext::sh -c touch% /tmp/pwned", "file:///etc", "/etc", "../x", "-x", "git://example.com/x.git", "http://example.com/x.git", ""} {
		cache := t.TempDir()
		src := contract.SharedSource{Name: "studio", URL: bad}
		_, err := git.New(localtest.NewRunner(t), local.New(nil, nil, nil, nil).FS, cache, src).Read(t.Context())
		if err == nil || !strings.Contains(err.Error(), "is no url") {
			t.Errorf("a source at %q read with %v", bad, err)
		}
	}
}

func TestAnHttpsSshOrShortUrlIsCloned(t *testing.T) {
	for _, good := range []string{"https://example.com/studio.git", "ssh://git@example.com/studio.git", "git@example.com:me/studio.git"} {
		cache := t.TempDir()
		dir := filepath.Join(cache, "sources", "studio")
		run := localtest.NewRunner(t)
		clones := run.On("git", quiet("clone", "--quiet", "--", good, dir)...).Answer(func() local.Output {
			clone(t, dir, nil)
			return local.Output{}
		})
		run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Reply(status(pinned))
		src := contract.SharedSource{Name: "studio", URL: good}
		if _, err := git.New(run, local.New(nil, nil, nil, nil).FS, cache, src).Read(t.Context()); err != nil {
			t.Errorf("%s: %v", good, err)
		}
		if clones.Seen != 1 {
			t.Errorf("%s cloned %d times, want once", good, clones.Seen)
		}
	}
}
