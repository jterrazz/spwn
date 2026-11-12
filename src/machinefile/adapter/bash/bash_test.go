package bash_test

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/adapter/bash"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

const lines = "mkdir -p ~/src\ncd ~/src && ls\n"

func TestBashKeepsTheContract(t *testing.T) {
	contracttest.Shell(t, lines, func(t *testing.T, fails bool) contract.Shell {
		r := localtest.NewRunner(t)
		out := local.Output{}
		if fails {
			out = local.Output{ExitCode: 128, Stderr: "fatal: repository '/mnt/src' does not exist\n"}
		}
		r.On("bash", "-e", "-c", lines).Reply(out)
		return bash.New(r, "/home/admin")
	})
}

func TestBashRunsTheLinesFromTheHomeAndBounded(t *testing.T) {
	// invariant: the lines start at the user's home, read nothing on stdin
	// and never run unbounded.
	r := localtest.NewRunner(t)
	r.On("bash", "-e", "-c", lines).Reply(local.Output{})
	if _, err := bash.New(r, "/home/admin").Run(context.Background(), lines); err != nil {
		t.Fatal(err)
	}
	if len(r.Calls) != 1 || r.Calls[0].Dir != "/home/admin" || r.Calls[0].Timeout <= 0 || r.Calls[0].Stdin != "" {
		t.Errorf("calls %+v; want one, from the home, bounded, with nothing on its stdin", r.Calls)
	}
}
