package agentsrecord

import (
	"context"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the terminal of the VM's multiplexer the hook runs in, empty outside one.
	Terminal string
}

type Input struct {
	App  string
	Args []string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: run by an app's hook on each of its agent's events: it prints
	// nothing and exits 0 within catalog.AgentRecordBudget whatever happens, so
	// a hook never blocks nor fails the agent, a flag it does not know or
	// cannot read included (exit 2 is the app's blocking code). The event is
	// the last argument when the app passes one, else stdin.
	cmd := kit.Command(kit.Leaf[Input]{
		Use:      "record",
		Short:    "Hand one event of an agent's hook to its journal",
		Plumbing: true,
		Args:     cobra.ArbitraryArgs,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.App, "app", "", "the app whose hook runs it")
		},
		Bind: func(in *Input, args []string) error {
			in.Args = args
			return nil
		},
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			ctx, cancel := context.WithTimeout(ctx, catalog.AgentRecordBudget)
			defer cancel()
			stdin, _ := o.Streams()
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() { _ = recover() }()
				payload, ok := payloadOf(in.Args, stdin)
				if !ok {
					return
				}
				_, _ = call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Record, agentscalls.RecordIn{App: agents.App(in.App), Terminal: d.Terminal, Payload: payload}, nil)
			}()
			select {
			case <-done:
			case <-ctx.Done():
			}
			return nil
		},
	}, env)
	cmd.FParseErrWhitelist.UnknownFlags = true
	cmd.SetFlagErrorFunc(func(*cobra.Command, error) error { return nil })
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		if pre := c.Root().PersistentPreRunE; pre != nil {
			_ = pre(c, args)
		}
		return nil
	}
	run := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		_ = run(c, args)
		return nil
	}
	return cmd
}

func payloadOf(args []string, stdin io.Reader) (string, bool) {
	if len(args) > 0 {
		return args[len(args)-1], true
	}
	if stdin == nil {
		return "", false
	}
	data, err := io.ReadAll(io.LimitReader(stdin, catalog.AgentPayloadCap))
	return string(data), err == nil && len(data) > 0
}
