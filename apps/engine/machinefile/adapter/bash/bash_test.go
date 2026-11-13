package bash_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/bash"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the real runner, each command it ran kept.
type kept struct {
	local.Runner
	calls []local.Cmd
}

func (k *kept) Output(ctx context.Context, c local.Cmd) (local.Output, error) {
	k.calls = append(k.calls, c)
	return k.Runner.Output(ctx, c)
}

// value: the owner's lines run by the bash this machine ships, from a home
// of the test's own.
func onBash(t *testing.T) (sh *bash.Bash, home string) {
	sh, home, _ = recorded(t)
	return sh, home
}

func recorded(t *testing.T) (sh *bash.Bash, home string, r *kept) {
	t.Helper()
	home = t.TempDir()
	r = &kept{Runner: local.New([]string{"HOME=" + home, "PATH=/usr/bin:/bin"}, nil, nil, nil).Run}
	return bash.New(r, home), home, r
}

func TestBashKeepsTheContract(t *testing.T) {
	contracttest.Shell(t, func(t *testing.T, _ bool) contract.Shell {
		sh, _ := onBash(t)
		return sh
	})
}

func TestBashStopsAtTheFirstLineThatFails(t *testing.T) {
	// invariant: a line ends the run when its status is not zero, wherever
	// -e alone would let it pass, and nothing after it runs; a multi-line
	// construct is one command, and the shell is one, so a cd and an export
	// carry to the lines after.
	cases := []struct {
		name, lines, said string
		exit              int
		ran               bool
	}{
		{name: "the left of &&", lines: "cd /nonexistent && echo never\ntouch ran\n", exit: 1, said: "station: exit 1 at: cd /nonexistent && echo never"},
		{name: "a pipe", lines: "false | cat\ntouch ran\n", exit: 1, said: "station: exit 1 at: false | cat"},
		{name: "a multi-line if", lines: "if [ -d / ]; then\n  cd /\n  test -d /nonexistent\nfi\ntouch ran\n", exit: 1, said: "station: exit 1 at: if [ -d / ]; then"},
		{name: "an exit", lines: "exit 7\ntouch ran\n", exit: 7, said: "station: exit 7 at: exit 7"},
		{name: "a line ending in CR", lines: "false\r\ntouch ran\r\n", exit: 1, said: "station: exit 1 at: false"},
		{name: "a comment ending in a backslash", lines: "  # a note \\\nfalse\ntouch ran\n", exit: 1, said: "station: exit 1 at: false"},
		{name: "a command reading the lines' descriptor", lines: "if { true <&3; } 2>/dev/null; then exit 9; fi\nexit 4\n", exit: 4, said: "station: exit 4 at: exit 4"},
		{name: "a test that guards", lines: "test -d /nonexistent || mkdir -p sub\ncd sub\nexport GOT=1\nif [ \"$GOT\" = 1 ]; then\n  cat <<END > ran\nheredoc\nEND\nfi\necho a \\\n  b\n", ran: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sh, home := onBash(t)
			got, err := sh.Run(context.Background(), c.lines)
			if err != nil || got.ExitCode != c.exit {
				t.Fatalf("Run = %+v, %v; want exit %d", got, err, c.exit)
			}
			if c.said != "" && !strings.HasSuffix(got.Stderr, c.said+"\n") {
				t.Errorf("stderr %q; want it to end %q", got.Stderr, c.said)
			}
			_, missing := os.ReadFile(filepath.Join(home, "ran"))
			if c.ran {
				_, missing = os.ReadFile(filepath.Join(home, "sub", "ran"))
			}
			if (missing == nil) != c.ran {
				t.Errorf("the line after ran: %v; want %v", missing == nil, c.ran)
			}
		})
	}
}

func TestBashHandsTheLinesAsAnArgumentFromTheHomeAndBounded(t *testing.T) {
	// invariant: the lines are the driver's argument, never its text; they
	// start at the user's home, read nothing on stdin and never run
	// unbounded.
	sh, home, r := recorded(t)
	const lines = "echo '$(touch pwned)' > said\n"
	if got, err := sh.Run(context.Background(), lines); err != nil || got.ExitCode != 0 {
		t.Fatalf("Run = %+v, %v; want exit 0", got, err)
	}
	if _, err := os.ReadFile(filepath.Join(home, "pwned")); err == nil {
		t.Error("the lines were read as the driver's own text")
	}
	if len(r.calls) != 1 {
		t.Fatalf("calls %+v; want one", r.calls)
	}
	c := r.calls[0]
	if c.Name != "bash" || len(c.Args) != 5 || c.Args[4] != lines || strings.Contains(c.Args[2], "pwned") {
		t.Errorf("argv %q; want bash -e -c <driver> station <lines>", c.Args)
	}
	if c.Dir != home || c.Timeout <= 0 || c.Stdin != "" {
		t.Errorf("call %+v; want it from the home, bounded, with nothing on its stdin", c)
	}
}
