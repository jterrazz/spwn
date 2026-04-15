package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/compile"
	_ "spwn.sh/packages/compile/runtimes/claudecode" // register the claude-code runtime
	"spwn.sh/packages/compile/source"
	"spwn.sh/packages/project"
)

func init() {
	compileCmd.Flags().StringVar(&compileRuntime, "runtime", "", "Target runtime (defaults to the runtime declared in agent.yaml, fallback: claude-code)")
	compileCmd.Flags().StringVar(&compileOut, "out", "dist", "Output directory for the compiled tree")
	compileCmd.Flags().StringVar(&compileAgent, "agent", "", "Compile only the named agent (filter the Tree)")
	compileCmd.Flags().StringVar(&compileWorld, "world", "", "World from spwn.yaml to compile (default: sole world)")
	compileCmd.Flags().BoolVar(&compileDryRun, "dry-run", false, "Print paths that would be written, don't touch disk")
	compileCmd.Flags().BoolVar(&compileJSON, "json", false, "Emit a machine-readable build report on stdout")
	compileCmd.Flags().BoolVar(&compileForce, "force", false, "Overwrite existing files in --out without prompting")
	rootCmd.AddCommand(compileCmd)
}

var (
	compileRuntime string
	compileOut     string
	compileAgent   string
	compileWorld   string
	compileDryRun  bool
	compileJSON    bool
	compileForce   bool
)

// compileReport is the CLI-owned JSON schema for `spwn compile --json`.
type compileReport struct {
	Runtime   string   `json:"runtime"`
	OutDir    string   `json:"outDir"`
	FileCount int      `json:"fileCount"`
	Paths     []string `json:"paths"`
}

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile the project into a runtime-specific file tree",
	Long: `Render the project through the claude-code runtime and materialise
the resulting Tree to disk.

Useful for previewing what spwn up would bake into its container,
debugging renderer output, and packaging for non-Docker runtimes.

  spwn compile                      # -> ./dist
  spwn compile --out ./preview
  spwn compile --dry-run            # list paths, touch nothing
  spwn compile --agent neo          # filter to one agent
  spwn compile --json               # machine-readable report`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if compileDryRun && compileJSON {
			return fmt.Errorf("--dry-run and --json are mutually exclusive")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve cwd: %w", err)
		}

		p, err := project.Find(cwd)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
		if p == nil {
			return fmt.Errorf(
				"no spwn.yaml found in %s or any parent directory.\nRun `spwn init` to create one",
				cwd)
		}

		src, err := source.Load(p.Root)
		if err != nil {
			return fmt.Errorf("load project source: %w", err)
		}

		runtimeName, err := source.ResolveRuntime(src, compileRuntime)
		if err != nil {
			return err
		}

		input, err := source.ToCompileInput(src, compileWorld)
		if err != nil {
			return err
		}

		// Fail loudly when an agent in the selected world has an
		// empty / missing AGENTS.md. The renderer itself doesn't
		// inspect AGENTS.md bytes yet, so without this guard the
		// user gets a successful compile for a silently-broken
		// agent. `spwn check --deep` reports the same finding.
		if err := requireAgentPrompts(src, input); err != nil {
			return err
		}

		tree, err := compile.Compile(runtimeName, input)
		if err != nil {
			// Surface the known runtime list so typos ("codex" ->
			// "claude-code") are self-correcting. Query the real
			// registry rather than hardcoding the list.
			if strings.Contains(err.Error(), "unknown runtime") {
				return fmt.Errorf(
					"%v\n\nKnown runtimes: %s", err,
					strings.Join(compile.RegisteredRuntimes(), ", "))
			}
			return fmt.Errorf("compile: %w", err)
		}

		if compileAgent != "" {
			filtered, known := filterTreeByAgent(tree, compileAgent)
			if len(filtered.Paths()) == 0 {
				return fmt.Errorf(
					"no entries match --agent %q; known agents in the Tree: %v",
					compileAgent, known)
			}
			tree = filtered
		}

		out := cmd.OutOrStdout()

		if compileDryRun {
			for _, p := range tree.Paths() {
				fmt.Fprintln(out, p)
			}
			return nil
		}

		absOut, err := filepath.Abs(compileOut)
		if err != nil {
			return fmt.Errorf("resolve --out: %w", err)
		}

		// Guard against clobbering a non-empty dir unless --force.
		if !compileForce {
			if nonEmpty, err := dirNonEmpty(absOut); err != nil {
				return fmt.Errorf("inspect --out: %w", err)
			} else if nonEmpty {
				return fmt.Errorf(
					"output directory %s is not empty; re-run with --force to overwrite",
					absOut)
			}
		} else if absOut != "" {
			// --force = replace the tree. Remove the existing
			// directory first so stale files from a prior compile
			// (or a prior `--agent` filter) don't mix with the new
			// output. Safety guard: refuse to wipe paths that look
			// like filesystem roots or the cwd itself.
			if err := safeRemoveOutDir(absOut); err != nil {
				return fmt.Errorf("clean --out: %w", err)
			}
		}

		if err := tree.WriteTo(absOut); err != nil {
			return fmt.Errorf("write tree: %w", err)
		}

		paths := tree.Paths()
		if compileJSON {
			report := compileReport{
				Runtime:   runtimeName,
				OutDir:    absOut,
				FileCount: len(paths),
				Paths:     paths,
			}
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		}

		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %s  %s\n", ui.Green("✓"), ui.Strong("Compile complete"))
		fmt.Fprintf(out, "     %s\n", ui.Faint(absOut))
		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %d file(s) written, runtime=%s\n", len(paths), runtimeName)
		fmt.Fprintln(out)
		return nil
	},
}

// filterTreeByAgent returns a new Tree containing only entries that
// belong to the named agent (prefix "agents/<name>/"), along with the
// sorted set of agent names observed in the input tree — useful for
// error messages when the filter drops everything.
func filterTreeByAgent(t *compile.Tree, name string) (*compile.Tree, []string) {
	out := compile.New()
	prefix := "agents/" + name + "/"
	for _, p := range t.Paths() {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		body, _ := t.Get(p)
		out.Add(p, body)
	}
	return out, listTreeAgents(t)
}

// listTreeAgents returns the sorted set of agent names present in the
// Tree (extracted from paths of shape "agents/<name>/...").
func listTreeAgents(t *compile.Tree) []string {
	seen := map[string]struct{}{}
	for _, p := range t.Paths() {
		if !strings.HasPrefix(p, "agents/") {
			continue
		}
		rest := strings.TrimPrefix(p, "agents/")
		if i := strings.IndexByte(rest, '/'); i > 0 {
			seen[rest[:i]] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	return out
}

// requireAgentPrompts validates that every agent selected for this
// compile run has a non-empty AGENTS.md on disk. Empty prompts would
// otherwise produce a silently-templated CLAUDE.md with no system
// instructions — worse than a loud error.
func requireAgentPrompts(src *source.ProjectSource, input compile.Input) error {
	if src == nil {
		return nil
	}
	byName := make(map[string]int, len(src.Agents))
	for i, a := range src.Agents {
		byName[a.Name] = i
	}
	var missing []string
	for _, a := range input.Agents {
		idx, ok := byName[a.Name]
		if !ok {
			continue
		}
		if len(strings.TrimSpace(string(src.Agents[idx].AgentMD))) == 0 {
			missing = append(missing, a.Name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"agent prompt is missing or empty for: %s\nCreate spwn/agents/<name>/AGENTS.md with the agent's system prompt",
		strings.Join(missing, ", "))
}

// safeRemoveOutDir removes dir if and only if it looks like a real,
// project-relative output directory. It refuses to touch filesystem
// roots, the user's home, and missing paths (which are a no-op). The
// guard is cheap insurance against a bogus --out value wiping more
// than the compile tree.
func safeRemoveOutDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("empty output directory")
	}
	clean := filepath.Clean(dir)
	if clean == "/" || clean == "." || clean == ".." {
		return fmt.Errorf("refusing to remove %q", clean)
	}
	// Refuse to touch $HOME exactly.
	if home, err := os.UserHomeDir(); err == nil && clean == filepath.Clean(home) {
		return fmt.Errorf("refusing to remove home directory %q", clean)
	}
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("--out %s is not a directory", clean)
	}
	return os.RemoveAll(clean)
}

func dirNonEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return len(entries) > 0, nil
}
