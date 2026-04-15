package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/catalog/examples"
	"spwn.sh/packages/project"
	"spwn.sh/packages/world"
	"spwn.sh/packages/paths"
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

const exampleRefPrefix = "@spwn/"

var initCmd = &cobra.Command{
	Use:   "init [example-ref]",
	Short: "Scaffold a spwn project in the current directory",
	Long: `Scaffold a spwn project in the current directory.

Without arguments, creates a blank spwn.yaml plus a default ./spwn/
tree (one world, one agent) and adds .spwn/ to .gitignore.

A positional example ref of the form @spwn/<slug> installs one of
the bundled examples into the current directory instead. Example:

    spwn init @spwn/matrix

Use --global to instead seed ~/.spwn/ with a world config (legacy
user-home mode, kept for backward compatibility).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if initGlobal {
			return runInitGlobal(cmd)
		}
		if len(args) == 1 {
			return runInitExample(cmd, args[0])
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

	// Reject invalid --name up-front so we never leave an unvalidated
	// spwn.yaml on disk that `spwn check` immediately rejects. Uses
	// the same regex the manifest enforces.
	if initName != "" && !project.IsValidProjectName(initName) {
		return fmt.Errorf("invalid --name %q — must match ^[a-z0-9][a-z0-9-]*$ (lowercase letters, digits, and dashes)", initName)
	}

	if err := project.Init(cwd, project.InitOpts{
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
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn.yaml                      # project manifest (worlds live inline)")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn.lock.yaml                 # committed package pins (like package-lock.json)")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn/agents/neo/               # starter agent")
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "  Gitignored:")
	fmt.Fprintln(cmd.OutOrStdout(), "    .spwn/                         # local state")
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "  Next:")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn auth                      # verify your credentials")
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn agent neo                 # open an interactive session with neo")
	s.Blank()

	return nil
}

// parseExampleRef validates a `@spwn/<slug>` argument and returns the
// bare slug. Anything else is a hard error with a one-line hint.
func parseExampleRef(ref string) (string, error) {
	if !strings.HasPrefix(ref, exampleRefPrefix) {
		return "", fmt.Errorf("example ref must start with %q (e.g. @spwn/matrix), got %q", exampleRefPrefix, ref)
	}
	slug := strings.TrimPrefix(ref, exampleRefPrefix)
	if slug == "" || strings.ContainsAny(slug, "/ \t") {
		return "", fmt.Errorf("invalid example slug in %q (expected @spwn/<slug>)", ref)
	}
	return slug, nil
}

func runInitExample(cmd *cobra.Command, ref string) error {
	if initName != "" {
		return fmt.Errorf("--name cannot be used with an example ref; it only applies to the blank scaffold")
	}

	slug, err := parseExampleRef(ref)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}

	// Honor --force: if the user passed it and a manifest already
	// exists, clear it so examples.Install can write fresh content
	// (examples.Install itself never overwrites).
	if initForce {
		if err := os.Remove(filepath.Join(cwd, "spwn.yaml")); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove existing spwn.yaml: %w", err)
		}
	}

	rep, err := examples.Install(slug, cwd)
	if err != nil {
		return fmt.Errorf("install example %s: %w", ref, err)
	}

	// Example installs ship a `.spwn/` runtime state dir at first
	// spawn; make sure `.gitignore` excludes it so users don't
	// accidentally commit local world state.
	if err := project.AppendGitignore(cwd); err != nil {
		return fmt.Errorf("update .gitignore: %w", err)
	}

	s := ui.New()
	s.Blank()
	s.Success(fmt.Sprintf("Installed example %s", ref))
	s.Blank()
	out := cmd.OutOrStdout()
	if rep.ManifestAdded {
		fmt.Fprintln(out, "  spwn.yaml          # created")
	} else {
		fmt.Fprintln(out, "  spwn.yaml          # kept existing (use --force to replace)")
	}
	if len(rep.WorldsAdded) > 0 {
		fmt.Fprintln(out, "  Worlds added:      "+strings.Join(rep.WorldsAdded, ", "))
	}
	if len(rep.WorldsSkipped) > 0 {
		fmt.Fprintln(out, "  Worlds skipped:    "+strings.Join(rep.WorldsSkipped, ", "))
	}
	if len(rep.AgentsAdded) > 0 {
		fmt.Fprintln(out, "  Agents added:      "+strings.Join(rep.AgentsAdded, ", "))
	}
	if len(rep.AgentsSkipped) > 0 {
		fmt.Fprintln(out, "  Agents skipped:    "+strings.Join(rep.AgentsSkipped, ", "))
	}
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  Next:")
	fmt.Fprintln(out, "    spwn up                        # spawn the world")
	s.Blank()

	return nil
}

func runInitGlobal(cmd *cobra.Command) error {
	s := ui.New()

	baseDir := paths.BaseDir()
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
