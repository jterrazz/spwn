package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/catalog"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/project"
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

const exampleRefPrefix = "spwn:"

var initCmd = &cobra.Command{
	Use:   "init [example-ref]",
	Short: "Scaffold a spwn project in the current directory",
	Long: `Scaffold a spwn project in the current directory.

Without arguments, creates a blank spwn.yaml plus a default ./spwn/
tree (one world, one agent) and adds .spwn/ to .gitignore.

A positional example ref installs one of the bundled gallery entries
into the current directory. Bare names resolve through the catalog:

    spwn init matrix          # shorthand for spwn init spwn:matrix
    spwn init spwn:matrix     # explicit form

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
	fmt.Fprintln(cmd.OutOrStdout(), "    spwn.lock                 # committed dependency pins (like package-lock.json)")
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

// parseExampleRef normalises an init argument to the bare slug. Accepts:
//   - "spwn:<slug>" — explicit form, slug extracted.
//   - "<slug>"      — bare form, resolved against the gallery
//     (catalog.ShippedSlugs) and rejected with a known-list hint when
//     no gallery entry matches.
//
// Anything else (uppercase, other schemes, legacy `@owner/name`) is
// rejected with the scheme grammar error.
func parseExampleRef(ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	resolved, err := dependency.ResolveCLI(trimmed, catalog.ShippedSlugs())
	if err != nil {
		return "", err
	}
	// After resolution the ref must be `spwn:<slug>` — `spwn init`
	// only accepts catalog gallery entries. Local-scheme refs
	// (skill:/tool:/hook:) and github: refs are not installable.
	if !strings.HasPrefix(resolved, exampleRefPrefix) {
		return "", fmt.Errorf("example ref must be a gallery entry (e.g. spwn:matrix), got %q", ref)
	}
	slug := strings.TrimPrefix(resolved, exampleRefPrefix)
	slug, _ = dependency.SplitVersion(slug)
	if slug == "" || strings.ContainsAny(slug, "/ \t") {
		return "", fmt.Errorf("invalid example slug in %q (expected spwn:<slug>)", ref)
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
	// exists, clear it so catalog.Install can write fresh content
	// (catalog.Install itself never overwrites).
	if initForce {
		if err := os.Remove(filepath.Join(cwd, "spwn.yaml")); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove existing spwn.yaml: %w", err)
		}
	}

	rep, err := catalog.Install(slug, cwd)
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
	s.Success(fmt.Sprintf("Installed example %s%s", exampleRefPrefix, slug))
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

	baseDir := platform.BaseDir()
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
