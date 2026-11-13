package guestserve

import (
	"context"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: serves station.api/v2 frames read from in, answered on out.
	Serve func(ctx context.Context, in io.Reader, out io.Writer) error
}

type Input struct{ Stdio bool }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: plumbing its Mac's engine runs inside the VM over its one
	// connection: the VM's own calls, answered as frames on stdout.
	return kit.Command(kit.Leaf[Input]{
		Use:      "serve",
		Short:    "Serve this VM's API to its Mac's engine",
		Plumbing: true,
		Declare:  func(f *pflag.FlagSet, in *Input) { f.BoolVar(&in.Stdio, "stdio", false, "over stdin and stdout") },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if !in.Stdio {
				return kit.Refuse("station guest serve speaks over stdio only", "station guest serve --stdio")
			}
			r, w := io.Streams()
			return d.Serve(ctx, r, w)
		},
	}, env)
}
