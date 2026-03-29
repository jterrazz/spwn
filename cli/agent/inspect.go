package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jterrazz/spwn/cli/ui"
	agentDomain "github.com/jterrazz/spwn/domains/agent"
	"github.com/jterrazz/spwn/shared/config"
	"github.com/spf13/cobra"
)

var inspectJSON bool

func init() {
	inspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "Output as JSON")
	Cmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <agent-name>",
	Short: "Show agent details, Mind layers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		info, err := agentDomain.InspectAgent(name)
		if err != nil {
			return fmt.Errorf("error: agent %q not found.\nRun 'spwn agent list' to see available agents.", name)
		}

		if inspectJSON {
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		s.Blank()
		s.Info("Agent:", info.Name)
		s.Info("Path:", info.Path)
		s.Blank()

		for _, layer := range config.MindLayers {
			files := info.Layers[layer]
			if len(files) == 0 {
				s.Info(layer+"/", "(empty)")
			} else {
				detail := fmt.Sprintf("%d file(s)", len(files))
				if len(files) <= 3 {
					detail = fmt.Sprintf("%d file(s)  (%s)", len(files), strings.Join(files, ", "))
				}
				s.Info(layer+"/", detail)
			}
		}

		// Show recent journal entries
		entries, err := agentDomain.ListJournal(info.Path, 5)
		if err == nil && len(entries) > 0 {
			s.Blank()
			for _, e := range entries {
				ts := e.CreatedAt.Format("2006-01-02 15:04")
				s.Info(ts, fmt.Sprintf("%-24s %-10s %s", e.UniverseID, e.Outcome, ui.FormatDuration(e.Duration)))
			}
		}

		s.Blank()

		return nil
	},
}
