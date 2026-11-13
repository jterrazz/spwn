package portsundeclare

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageports "github.com/jterrazz/studio-station/apps/engine/pages/ports"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ Machine, Port string }

func (in *Input) name() string { return strings.TrimPrefix(in.Port+" on "+in.Machine, " on ") }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "undeclare <machine> <port>",
		Short:   "Take back a port's declaration: it is red again while it listens where something outside " + catalog.TailnetProduct + " reaches it",
		Example: "  station ports undeclare jterrazz-mac-mini 5000",
		Args:    cobra.RangeArgs(1, 2),
		Bind: func(in *Input, args []string) error {
			in.Machine, in.Port = "", args[len(args)-1]
			if len(args) == 2 {
				in.Machine = args[0]
			}
			return nil
		},
		Asks: true,
		Verb: func(in *Input) string { return "undeclare " + in.name() },
		Confirm: func(in *Input) string {
			return "Undeclare " + in.name() + "? it is red again while it is reachable outside " + catalog.TailnetProduct
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var out portscalls.UndeclareOut
			err := io.Confirmed(func(answer string) (err error) {
				out, err = call.Invoke(ctx, env.Calls, env.Target(), portscalls.Undeclare, portscalls.UndeclareIn{Machine: in.Machine, Port: in.Port, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pageports.Undeclared(out.Declaration, d.ThisMachine), nil)
		},
	}, env)
}
