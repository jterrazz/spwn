package imagescmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/images"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageimages "github.com/jterrazz/studio-station/apps/engine/pages/images"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the images the catalog knows, one no Mac holds shown as such.
	Images []catalog.Image
	// value: the images of every Mac, this one first; nil reads the target's alone.
	Every func(ctx context.Context) ([]images.Report, error)
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[struct{}]{
		Use:     "images",
		Short:   "The VM images of every Mac, with their versions and the VMs made from each",
		Example: "  station images\n  station images --json",
		Group:   kit.ImagesAndEngine,
		Read:    func(*struct{}) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			var reports []images.Report
			var err error
			if d.Every != nil && !env.Far() {
				reports, err = d.Every(ctx)
			} else {
				reports, err = call.Invoke(ctx, env.Calls, env.Target(), imagescalls.List, imagescalls.ListIn{}, nil)
			}
			if err != nil {
				return err
			}
			return io.Page(pageimages.List(reports, d.Images, d.ThisMachine), pageimages.ListContract(reports))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
