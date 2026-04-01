package msg

import (
	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
)

var defaultMsgHelp func(*cobra.Command, []string)

func init() {
	defaultMsgHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(msgHelp)
}

func msgHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "msg" {
		if defaultMsgHelp != nil {
			defaultMsgHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ msg")+" "+ui.Faint("— inter-agent messaging"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "send <to> --from <from> \"text\"", Desc: "Send a message"},
				{Name: "inbox <name>", Desc: "Show inbox messages"},
				{Name: "watch <name>", Desc: "Watch for new messages"},
			}},
		},
		"spwn msg [command]",
		"Use \"spwn msg <command> --help\" for more information.",
	)
}

// Cmd is the msg command group.
var Cmd = &cobra.Command{
	Use:   "msg",
	Short: "Agent messaging — send, inbox, watch",
	Long:  `Send and receive messages between agents across worlds.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
