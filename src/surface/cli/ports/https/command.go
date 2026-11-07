package portshttps

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageports "github.com/jterrazz/studio-station/src/pages/ports"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ On bool }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "https <on|off>",
		Short: "Turn HTTPS links on or off on a Mac: on, an HTTP server's share gets an https:// link with a valid certificate; off, an http:// one",
		Example: "  station ports https off\n" +
			"  station ports https on --on mac-mini",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"on", "off"},
		Bind: func(in *Input, args []string) error {
			in.On = args[0] == "on"
			return nil
		},
		Read: func(*Input) bool { return false },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, env.Target(), portscalls.SetHTTPS, portscalls.SetHTTPSIn{On: in.On}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageports.Links(out.HTTPS, d.ThisMachine), nil)
		},
	}, env)
}
