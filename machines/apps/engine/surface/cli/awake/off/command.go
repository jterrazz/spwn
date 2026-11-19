package awakeoff

import (
	"context"

	"github.com/spf13/cobra"

	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Label       func(mac string) string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[struct{}]{
		Use:     "off",
		Short:   "End the keep-awake now: the Mac sleeps normally again; Never sleeps stays",
		Example: "  station awake off\n  station awake off --on mac-mini",
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			s, err := call.Invoke(ctx, env.Calls, env.Target(), awakecalls.Off, awakecalls.OffIn{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageawake.Off(s, d.Label(s.Mac), d.ThisMachine), s)
		},
	}, env)
}
