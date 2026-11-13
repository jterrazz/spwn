package imagesbuild

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageimages "github.com/jterrazz/studio-station/apps/engine/pages/images"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the images the catalog knows, by name.
	Images []catalog.Image
}

type Input struct {
	Names  []string
	Binary string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:       "build [<image>...]",
		Short:     "Build images; bare, every image that is missing or whose recipe changed",
		Example:   "  station images build\n  station images build station-linux --binary .artifacts/go/station-linux-arm64",
		Args:      cobra.ArbitraryArgs,
		ValidArgs: imageNames(d.Images),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Binary, "binary", "", "the linux station binary to put in the image (a dev build needs one: make build-linux)")
		},
		Bind: func(in *Input, args []string) error {
			in.Names = args
			return nil
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			steps := kit.Steps(io)
			records, err := call.Invoke(ctx, env.Calls, env.Target(), imagescalls.Build, imagescalls.BuildIn{Names: in.Names, Binary: in.Binary}, steps)
			steps.Stop()
			if err != nil {
				return err
			}
			return io.Page(pageimages.Built(records, d.ThisMachine), nil)
		},
	}, env)
}

func imageNames(known []catalog.Image) []string {
	names := make([]string, 0, len(known))
	for _, img := range known {
		names = append(names, img.Name)
	}
	return names
}
