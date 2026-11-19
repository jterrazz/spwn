package machinespipe

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Pipe interface {
	// value: every refusal, before the first byte.
	Check(ctx context.Context, vm string) error
	Pipe(ctx context.Context, vm string, in io.Reader, out io.Writer) error
}

type Deps struct{ Machines Pipe }

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: plumbing for ssh's ProxyCommand: stdout is the byte stream, so
	// every refusal comes before the first byte and nothing else is printed.
	return kit.Command(kit.Leaf[Input]{
		Use:      "pipe <vm>",
		Short:    "Carry ssh's bytes to a VM of this Mac",
		Plumbing: true,
		Args:     cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Validate: func(ctx context.Context, in *Input) error {
			return d.Machines.Check(ctx, in.Name)
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			stdin, stdout := io.Streams()
			return d.Machines.Pipe(ctx, in.Name, stdin, stdout)
		},
	}, env)
}
