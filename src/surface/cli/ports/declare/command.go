package portsdeclare

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageports "github.com/jterrazz/studio-station/src/pages/ports"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ Machine, Port, Via, URL string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "declare <machine> <port>",
		Short: "Declare a port public: it is meant to be reached from outside " + catalog.TailnetProduct + ", by the route --via names (" + catalog.RouteNames() + ")",
		Example: "  station ports declare jterrazz-mac-mini 5000 --via lan\n" +
			"  station ports declare infra 443 --via cloudflare-tunnel --url https://jterrazz.com",
		Args: cobra.RangeArgs(1, 2),
		Bind: func(in *Input, args []string) error {
			in.Machine, in.Port = "", args[len(args)-1]
			if len(args) == 2 {
				in.Machine = args[0]
			}
			return nil
		},
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Via, "via", "", "the route that reaches it: "+catalog.RouteNames())
			f.StringVar(&in.URL, "url", "", "the link it is reached by from outside, opened and shown as its QR")
		},
		Read: func(*Input) bool { return false },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			out, err := call.Invoke(ctx, env.Calls, env.Target(), portscalls.Declare, portscalls.DeclareIn{Machine: in.Machine, Port: in.Port, Via: in.Via, URL: in.URL}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageports.Declared(out.Declaration, d.ThisMachine), nil)
		},
	}, env)
}
