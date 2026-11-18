package engineattach

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: serves station.api/v2 frames read from in, answered on out.
	Serve func(ctx context.Context, in io.Reader, out io.Writer) error
}

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: plumbing another Mac runs over ssh: its stdin and stdout are
	// that Mac's frames, so nothing else of this run writes to stdout.
	return kit.Command(kit.Leaf[struct{}]{
		Use:      "attach",
		Short:    "Serve this Mac's API over stdin and stdout, for another Mac",
		Plumbing: true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			in, out := io.Streams()
			return d.Serve(ctx, in, out)
		},
	}, env)
}
