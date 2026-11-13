package kit

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type On struct {
	Machine string
	// value: the machine this terminal uses, when no --on names one.
	Used        string
	ThisMachine string
	Refresh     func(ctx context.Context) error
	// value: whether a machine of that name exists; an error when it cannot
	// be told (a Mac that does not answer).
	Known func(ctx context.Context, name string) (bool, error)
}

// invariant: the remote's answer is the outcome; a refresh that fails leaves the
// blocks the next bare `station machines` rewrites, so it is no failure.
func (o *On) refresh(ctx context.Context, path string) {
	if o.Refresh == nil {
		return
	}
	switch path {
	case "machines create", "machines delete", "machines add", "machines forget":
		_ = o.Refresh(ctx)
	}
}

// value: the name any command accepts for the Mac it runs on (SPEC §2.5).
const thisMac = "this-mac"

func (o *On) Target() (string, bool) {
	if o == nil {
		return "", false
	}
	name := o.Machine
	if name == "" {
		name = o.Used
	}
	if name == "" || name == o.ThisMachine || name == thisMac {
		return "", false
	}
	return name, true
}

func (e Env) Target() machine.Ref {
	// invariant: --on is a target of the Router: the leaf runs here, its call
	// runs on that machine and its answer is rendered by this binary.
	if name, ok := e.On.Target(); ok && name != e.Self.Name {
		return machine.Ref{Name: name}
	}
	return e.Self
}

func (e Env) Far() bool { return e.Target() != e.Self }

const (
	// value: alias of v2's `station daemon`, the engine run under its old
	// name for one release.
	RetiredEngineCommand = "daemon"
)

func refusesOn(path string) bool {
	fields := strings.Fields(path)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "shell", "completion", "help", RetiredEngineCommand, "guest", "use":
		return true
	}
	return len(fields) > 1 && (fields[0] == "machines" && (fields[1] == "supervise" || fields[1] == "pipe") ||
		fields[0] == "engine" && (fields[1] == "run" || fields[1] == "attach"))
}

func RefuseOn(env Env) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if env.On == nil {
			return nil
		}
		if env.On.Machine == "" && c.DisableFlagParsing {
			env.On.Machine = rawOn(args)
		}
		if env.On.Machine == "" {
			return nil
		}
		p := path(c)
		if refusesOn(p) {
			return usage("--on does not apply to \""+p+"\"", words.RunOnItself(env.On.Machine))
		}
		if env.Far() && c.Annotations[owned] == "" && c.HasParent() {
			return usage("--on does not apply to \""+p+"\"", words.RunOnItself(env.On.Machine))
		}
		return nil
	}
}

// invariant: a command kit built answers --on with its calls; one it did not
// (fang's man) has none to send, so --on is refused for it.
const owned = "station.kit"

func rawOn(args []string) string {
	for i, a := range args {
		if a == "--" {
			return ""
		}
		if v, ok := strings.CutPrefix(a, "--on="); ok {
			return v
		}
		if a == "--on" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
