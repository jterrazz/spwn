package imagesupdate

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	imagescalls "github.com/jterrazz/studio-station/src/images/calls"
	"github.com/jterrazz/studio-station/src/page"
	pageimages "github.com/jterrazz/studio-station/src/pages/images"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the images the catalog knows, by name.
	Images []catalog.Image
}

type Input struct{ Names []string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:       "update [<image>...]",
		Short:     "Fetch the newest released version of images; bare, of every image held here",
		Example:   "  station images update\n  station images update linux",
		Args:      cobra.ArbitraryArgs,
		ValidArgs: imageNames(d.Images),
		Bind: func(in *Input, args []string) error {
			in.Names = args
			return nil
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			steps := kit.Steps(io)
			updates, err := call.Invoke(ctx, env.Calls, env.Target(), imagescalls.Update, imagescalls.UpdateIn{Names: in.Names}, steps)
			steps.Stop()
			if err != nil {
				return err
			}
			return io.Page(pageimages.Updated(updates, d.ThisMachine), nil)
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
