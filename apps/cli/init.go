package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/foundation"
	"spwn.sh/packages/manifest"
	"spwn.sh/packages/world"
)

func init() {
	initCmd.Flags().BoolVar(&initGlobal, "global", false, "Initialise ~/.spwn/ (legacy user-home mode)")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing spwn.yaml")
	initCmd.Flags().StringVar(&initName, "name", "", "Project name (default: current directory name)")
	rootCmd.AddCommand(initCmd)
}

var (
	initGlobal bool
	initForce  bool
	initName   string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a spwn project in the current directory",
	Long: `Scaffold a spwn project in the current directory.

Creates spwn.yaml and a committed ./spwn/ tree containing a default
world and a default agent. Adds .spwn/ to .gitignore for local state.

Use --global to instead seed ~/.spwn/ with a world config (legacy
user-home mode, kept for backward compatibility).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if initGlobal {
			return runInitGlobal(cmd)
		}
		return runInitLocal(cmd)
	},
}

func runInitLocal(cmd *cobra.Command) error {
	s := ui.New()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}

	if err := manifest.Init(cwd, manifest.InitOpts{
		Name:  initName,
		Force: initForce,
	}); err != nil {
		return err
	}

	name := initName
	if name == "" {
		name = filepath.Base(cwd)
	}

	s.Blank()
	s.Success("Initialised spwn project " + name)
	s.Blank()
	fmt.Fprintln(cmd.OutOrStdout(), "  Committed:")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn.yaml                      # project manifest")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn/agents/default/           # starter agent")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn/worlds/default.yaml       # starter world")
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "  Gitignored:")
	fmt.Fprintln(cmd.OutOrStdout(), "    .spwn/                         # local state")
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "  Next:")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn check                     # validate the project tree")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn up                        # spawn the default world")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn agent new <name>          # add another agent")
	s.Blank()

	return nil
}

func runInitGlobal(cmd *cobra.Command) error {
	s := ui.New()

	baseDir := foundation.BaseDir()
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", baseDir, err)
	}

	s.Blank()
	if err := world.CreateDefaultConfig(); err != nil {
		s.Log("default.yaml already exists, skipping")
	} else {
		s.Done("Created config", "default.yaml")
	}
	s.Blank()
	s.Success("Ready")
	s.Blank()

	return nil
}
