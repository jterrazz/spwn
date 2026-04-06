package agent

import (
	"fmt"
	"os"
	"path/filepath"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"github.com/spf13/cobra"
)

func init() {
	// Removed: now handled by `spwn profile <name>`
}

var mindCmd = &cobra.Command{
	Use:   "mind <agent-name>",
	Short: "Show the agent's Mind directory tree with file sizes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		if err := agentDomain.ValidateMind(name); err != nil {
			return fmt.Errorf("agent %q not found", name)
		}

		info, err := agentDomain.InspectAgent(name)
		if err != nil {
			return fmt.Errorf("cannot inspect agent: %w", err)
		}

		s.Blank()
		s.Info("Mind:", name)
		s.Blank()

		for _, layer := range foundation.MindLayers {
			files := info.Layers[layer]
			if len(files) == 0 {
				fmt.Fprintf(os.Stderr, "  %s/\n", layer)
				fmt.Fprintf(os.Stderr, "    (empty)\n")
			} else {
				fmt.Fprintf(os.Stderr, "  %s/\n", layer)
				for _, f := range files {
					fpath := filepath.Join(info.Path, layer, f)
					fi, err := os.Stat(fpath)
					if err == nil {
						fmt.Fprintf(os.Stderr, "    %-36s (%s)\n", f, formatSize(fi.Size()))
					} else {
						fmt.Fprintf(os.Stderr, "    %s\n", f)
					}
				}
			}
			fmt.Fprintln(os.Stderr)
		}

		return nil
	},
}
