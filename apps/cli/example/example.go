// Package example wires the `spwn example …` cobra tree. It's a
// tiny shell around the spwn.sh/examples package — listing and
// installing is all one-shot filesystem work, no state involved.
package example

import (
	"encoding/json"
	"fmt"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/examples"

	"github.com/spf13/cobra"
)

// Cmd is the parent command registered on the root in root.go.
var Cmd = &cobra.Command{
	Use:   "example",
	Short: "Install ready-made world + agent templates",
	Long: `Ship-in-a-binary gallery of pre-written templates for first-time
users and people who want a known-good starting point.

Each example is a complete world + one-or-more agents with profiles
already filled in. Installing copies the files into ~/.spwn/ without
touching anything that already exists, so it is safe to re-run.

Examples:
  spwn example list
  spwn example install matrix
  spwn example install paperclip-factory`,
}

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show every bundled example",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := examples.List()
		if err != nil {
			return err
		}

		// --json is the stable contract used by bundling tests and by
		// the Tauri pre-release verification hook. On any zero-list
		// result it still emits {"examples":[]} and exits non-zero so
		// CI can fail loudly instead of silently shipping a hollow
		// binary.
		if listJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(map[string]interface{}{"examples": list}); err != nil {
				return err
			}
			if len(list) == 0 {
				return fmt.Errorf("no examples bundled in this build")
			}
			return nil
		}

		if len(list) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no examples bundled in this build")
			return nil
		}
		s := ui.New(false, false, false)
		s.Blank()
		for _, ex := range list {
			s.Info(ex.Slug, ex.Tagline)
		}
		s.Blank()
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Install with: spwn example install <slug>"))
		s.Blank()
		return nil
	},
}

var installCmd = &cobra.Command{
	Use:   "install <slug>",
	Short: "Copy an example's worlds + agents into ~/.spwn/",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		s := ui.New(false, false, false)

		s.Blank()
		s.Start(fmt.Sprintf("Installing example %q...", slug))

		rep, err := examples.InstallInto(slug)
		if err != nil {
			if err == examples.ErrNotFound {
				return s.FailHint("Install failed", err,
					"Run 'spwn example list' to see valid slugs")
			}
			return s.FailHint("Install failed", err, "")
		}

		for _, w := range rep.WorldsAdded {
			s.Done("World added", w)
		}
		for _, w := range rep.WorldsSkipped {
			s.Info("World kept", w+" (already exists)")
		}
		for _, a := range rep.AgentsAdded {
			s.Done("Agent added", a)
		}
		for _, a := range rep.AgentsSkipped {
			s.Info("Agent kept", a+" (already exists)")
		}

		s.Blank()
		if len(rep.AgentsAdded) == 0 && len(rep.WorldsAdded) == 0 {
			s.Info("Everything already installed", "nothing to do")
		} else {
			s.Success(fmt.Sprintf("Example %q ready", slug))
		}
		s.Blank()
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "emit the bundled example list as JSON and fail with exit code 1 if empty")
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(installCmd)
}
