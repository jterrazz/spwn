package imagesdelete

import (
	"context"
	"strings"

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
	// value: the images the catalog knows, by name.
	Images []catalog.Image
}

type Input struct{ Image, Version string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "delete <image>[@<version>]",
		Short:   "Delete one version of an image, or every version; the VMs made from it stay",
		Example: "  station images delete station-linux@2026.11.01 --yes\n  station images delete station-linux --yes",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Image, in.Version, _ = strings.Cut(args[0], "@")
			return nil
		},
		Complete: func([]string, string) []string {
			names := make([]string, 0, len(d.Images))
			for _, img := range d.Images {
				names = append(names, img.Name)
			}
			return names
		},
		Asks: true,
		Confirm: func(in *Input) string {
			if in.Version == "" {
				return "Delete every version of " + in.Image + "?"
			}
			return "Delete " + in.Image + "@" + in.Version + "?"
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var removed []images.Record
			err := io.Confirmed(func(answer string) (err error) {
				removed, err = call.Invoke(ctx, env.Calls, env.Target(), imagescalls.Delete, imagescalls.DeleteIn{Image: in.Image, Version: in.Version, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pageimages.Deleted(removed, d.ThisMachine), nil)
		},
	}, env)
}
