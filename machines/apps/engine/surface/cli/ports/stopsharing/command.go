package portsstopsharing

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageports "github.com/jterrazz/studio-station/apps/engine/pages/ports"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ VM, Port string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "stop-sharing <vm> <port>",
		Short: "Stop sharing a VM's port: its reserved port is released, and it is not shared again until you share it; inside a VM, the port alone",
		Example: "  station ports stop-sharing linux-1 7000\n" +
			"  station ports stop-sharing 7000            # inside a VM",
		Args: cobra.RangeArgs(1, 2),
		Bind: func(in *Input, args []string) error {
			in.VM, in.Port = "", args[len(args)-1]
			if len(args) == 2 {
				in.VM = args[0]
			} else if vm, port, ok := strings.Cut(in.Port, ":"); ok {
				in.VM, in.Port = vm, port
			}
			return nil
		},
		Asks: true,
		Verb: func(in *Input) string { return "stop sharing " + strings.TrimPrefix(in.VM+":"+in.Port, ":") },
		Confirm: func(in *Input) string {
			return "Stop sharing " + strings.TrimPrefix(in.VM+":"+in.Port, ":") + "? it is not shared again when it listens, until you share it"
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var out portscalls.StopSharingOut
			err := io.Confirmed(func(answer string) (err error) {
				out, err = call.Invoke(ctx, env.Calls, env.Target(), portscalls.StopSharing, portscalls.StopSharingIn{VM: in.VM, Port: in.Port, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pageports.Stopped(out.Reservation, d.ThisMachine), nil)
		},
	}, env)
}
