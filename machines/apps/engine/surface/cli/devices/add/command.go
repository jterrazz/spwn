package devicesadd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagedevices "github.com/jterrazz/studio-station/apps/engine/pages/devices"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct {
	Name     string
	Key      string
	Channels []string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "add <name> <public key>",
		Short:   "Let another device reach this Mac and its VMs, by its public key; asks you to confirm at this Mac",
		Example: "  station devices add laptop ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGmY0fN5lE8XqXhcG0OB1oCw0SYqC0f6M1bM1jR5e9wz laptop@home\n  station devices add phone ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGmY0fN5lE8XqXhcG0OB1oCw0SYqC0f6M1bM1jR5e9wz --channels pipe",
		Args:    cobra.MinimumNArgs(3),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringSliceVar(&in.Channels, "channels", nil, "what it may open: "+channelWords()+"; all of them when left out")
		},
		Bind: func(in *Input, args []string) error {
			// invariant: the key is the rest of the line, as a .pub file holds
			// it: its type, its base64, then any comment.
			in.Name, in.Key = args[0], strings.Join(args[1:], " ")
			return nil
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			row, err := call.Invoke(ctx, env.Calls, env.Target(), devicescalls.Add, devicescalls.AddIn{Name: in.Name, Key: in.Key, Channels: in.Channels}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagedevices.Added(row, env.Target().Name, d.ThisMachine), row)
		},
	}, env)
}

func channelWords() string {
	words := make([]string, 0, len(devices.AllChannels()))
	for _, c := range devices.AllChannels() {
		words = append(words, string(c))
	}
	return strings.Join(words, ", ")
}
