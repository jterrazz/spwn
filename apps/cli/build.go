package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/runtimeres"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/transpile"
	_ "spwn.sh/packages/runtimes/defaults" // register every built-in runtime
	"spwn.sh/packages/transpile/source"
	"spwn.sh/packages/compile"
	"spwn.sh/packages/project"
)

func init() {
	// ── What to build ─────────────────────────────────────────
	buildCmd.Flags().StringVar(&buildRuntime, "runtime", "", "Target runtime. Defaults to the runtime declared in agent.yaml (fallback: claude-code)")
	buildCmd.Flags().StringVar(&buildWorld, "world", "", "World from spwn.yaml to build (required for multi-world projects)")
	buildCmd.Flags().StringVar(&buildAgent, "agent", "", "Compile only the named agent (tree-only mode)")

	// ── Image-mode flags ──────────────────────────────────────
	buildCmd.Flags().StringVar(&buildTag, "tag", "", "Image tag (default: spwn-<project>:latest)")
	buildCmd.Flags().StringVar(&buildBase, "base", "", "Base image to derive from (default: $SPWN_BASE_IMAGE, else spwn-world:latest)")
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Disable Docker build cache")

	// ── Tree-only mode flags ──────────────────────────────────
	buildCmd.Flags().BoolVar(&buildTreeOnly, "tree-only", false, "Stop after the compile step; write the Tree to --output instead of building a Docker image")
	buildCmd.Flags().StringVarP(&buildOut, "output", "o", "dist", "Output directory for --tree-only mode")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "Print paths that would be written, don't touch disk (requires --tree-only)")
	buildCmd.Flags().BoolVar(&buildForce, "force", false, "Overwrite existing files in --output without prompting (requires --tree-only)")

	// ── Output ────────────────────────────────────────────────
	buildCmd.Flags().BoolVar(&buildJSON, "json", false, "Emit a machine-readable build report on stdout")

	rootCmd.AddCommand(buildCmd)
}

var (
	buildRuntime  string
	buildWorld    string
	buildAgent    string
	buildTag      string
	buildBase     string
	buildNoCache  bool
	buildTreeOnly bool
	buildOut      string
	buildDryRun   bool
	buildForce    bool
	buildJSON     bool
)

// buildReport is the CLI-owned JSON schema for `spwn build --json`.
// Tree-only runs populate the TreeOnly + OutDir + Paths fields and
// leave image fields empty; image runs do the reverse.
type buildReport struct {
	Runtime   string   `json:"runtime"`
	World     string   `json:"world"`
	TreeFiles int      `json:"treeFiles"`
	TreeOnly  bool     `json:"treeOnly"`
	// Tree-only fields
	OutDir string   `json:"outDir,omitempty"`
	Paths  []string `json:"paths,omitempty"`
	// Image-mode fields
	Tag       string `json:"tag,omitempty"`
	ImageID   string `json:"imageId,omitempty"`
	BaseImage string `json:"baseImage,omitempty"`
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Transpile the project and compile it into a Docker image",
	Long: `Transpile the project with the target runtime (default: claude-code)
and compile the result into a derived Docker compile.

The image is FROM spwn-world:latest by default, with the transpiled
tree COPY'd to /world/. The resulting image carries the project's
name and the runtime name as Docker labels, so it's push-ready and
reproducible.

Pass --tree-only to stop after the transpile step and write the
generated file tree to --output (default: ./dist). No Docker
required, useful for previewing renderer output or authoring a
new runtime backend.

Use 'spwn up' to spawn a world from the current project. Use 'spwn
check --deep' to run the transpile dry-run as part of validation.

Examples:
  spwn build                                  # transpile + image, tag spwn-<project>:latest
  spwn build --tag spwn-myproj:v1
  spwn build --base spwn-world:2.1
  spwn build --runtime claude-code
  spwn build --world <name>                   # multi-world projects
  spwn build --no-cache
  spwn build --json
  spwn build --tree-only                      # transpile only, write to ./dist
  spwn build --tree-only --output ./preview
  spwn build --tree-only --dry-run            # list paths, touch nothing
  spwn build --tree-only --agent neo          # filter to one agent`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Flag combination guardrails. Keep these tight so users
		// don't silently pass a tree-only flag in image mode and
		// wonder why it had no effect.
		if buildDryRun && !buildTreeOnly {
			return fmt.Errorf("--dry-run requires --tree-only")
		}
		if buildDryRun && buildJSON {
			return fmt.Errorf("--dry-run and --json are mutually exclusive")
		}
		if buildAgent != "" && !buildTreeOnly {
			return fmt.Errorf("--agent filtering is only meaningful with --tree-only; the Docker image compiles the whole world")
		}
		if buildTreeOnly {
			if buildTag != "" {
				return fmt.Errorf("--tag is not meaningful with --tree-only (no image is built)")
			}
			if buildBase != "" {
				return fmt.Errorf("--base is not meaningful with --tree-only (no image is built)")
			}
			if buildNoCache {
				return fmt.Errorf("--no-cache is not meaningful with --tree-only (no image is built)")
			}
		}

		p, err := cliproject.Require()
		if err != nil {
			return err
		}

		// Validate before touching Docker - same rules as `spwn
		// check`. This keeps bad manifests from turning into
		// confusing docker build errors downstream. Skipped in
		// tree-only mode because the whole point of that path is
		// "render what you have, regardless of validation state".
		if !buildTreeOnly {
			issues := project.Validate(p, project.ValidateOpts{
				BuiltinTools:      catalogToolNames(),
				SupportedRuntimes: supportedRuntimes(),
			})
			if project.HasErrors(issues) {
				return fmt.Errorf("project has validation errors - run `spwn check` to see them")
			}
		}

		src, err := source.Load(p.Root)
		if err != nil {
			return fmt.Errorf("load project source: %w", err)
		}

		// Resolve runtime: --runtime override > agent declaration >
		// single authenticated provider > claude-code fallback. See
		// runtimeres.Resolve for the full precedence cascade; it
		// wraps source.ResolveRuntime with auth-state awareness so a
		// scaffold without a pinned backend picks the right runtime
		// for the logged-in user.
		runtimeName, err := runtimeres.Resolve(src, buildRuntime)
		if err != nil {
			return err
		}

		input, err := source.ToCompileInput(src, buildWorld)
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

		tree, err := transpile.Compile(runtimeName, input)
		if err != nil {
			// Surface the known runtime list so typos ("codex" ->
			// "claude-code") are self-correcting. Query the real
			// registry rather than hardcoding the list.
			if strings.Contains(err.Error(), "unknown runtime") {
				return fmt.Errorf(
					"%v\n\nKnown runtimes: %s", err,
					strings.Join(transpile.RegisteredRuntimes(), ", "))
			}
			return fmt.Errorf("compile: %w", err)
		}

		if buildAgent != "" {
			filtered, known := filterTreeByAgent(tree, buildAgent)
			if len(filtered.Paths()) == 0 {
				return fmt.Errorf(
					"no entries match --agent %q; known agents in the Tree: %v",
					buildAgent, known)
			}
			tree = filtered
		}

		if buildTreeOnly {
			return runBuildTreeOnly(cmd, runtimeName, input.WorldID, tree)
		}
		return runBuildImage(cmd, p, runtimeName, input.WorldID, tree)
	},
}

// runBuildTreeOnly materialises the compiled Tree to disk. This is
// the path formerly known as `spwn compile`: pure, no Docker,
// useful for previewing renderer output and writing new runtime
// backends.
func runBuildTreeOnly(cmd *cobra.Command, runtimeName, worldID string, tree *transpile.Tree) error {
	out := cmd.OutOrStdout()

	if buildDryRun {
		for _, p := range tree.Paths() {
			fmt.Fprintln(out, p)
		}
		return nil
	}

	absOut, err := filepath.Abs(buildOut)
	if err != nil {
		return fmt.Errorf("resolve --output: %w", err)
	}

	// Guard against clobbering a non-empty dir unless --force.
	if !buildForce {
		if nonEmpty, err := dirNonEmpty(absOut); err != nil {
			return fmt.Errorf("inspect --output: %w", err)
		} else if nonEmpty {
			return fmt.Errorf(
				"output directory %s is not empty; re-run with --force to overwrite",
				absOut)
		}
	} else if absOut != "" {
		// --force = replace the tree. Remove the existing directory
		// first so stale files from a prior build (or a prior
		// --agent filter) don't mix with the new output.
		if err := safeRemoveOutDir(absOut); err != nil {
			return fmt.Errorf("clean --output: %w", err)
		}
	}

	if err := tree.WriteTo(absOut); err != nil {
		return fmt.Errorf("write tree: %w", err)
	}

	paths := tree.Paths()
	if buildJSON {
		report := buildReport{
			Runtime:   runtimeName,
			World:     worldID,
			TreeFiles: len(paths),
			TreeOnly:  true,
			OutDir:    absOut,
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
}

// runBuildImage compiles the tree into a derived Docker compile.
// The project's name becomes the default tag, labels identify the
// runtime and world, and the base image defaults to
// spwn-world:latest so the result is layered on top of whatever
// the user's `spwn up` would have spawned from.
func runBuildImage(cmd *cobra.Command, p *project.Project, runtimeName, worldID string, tree *transpile.Tree) error {
	// Compute the image tag: explicit flag wins, otherwise
	// spwn-<project>:latest.
	tag := buildTag
	if tag == "" {
		tag = fmt.Sprintf("spwn-%s:latest", p.Manifest.Name)
	}

	// Resolve the base image: --base > $SPWN_BASE_IMAGE >
	// spwn-world:latest. This mirrors how spawn discovers the
	// base image, so e2e tests pinning SPWN_BASE_IMAGE don't
	// need an extra flag.
	baseImage := buildBase
	if baseImage == "" {
		if env := os.Getenv("SPWN_BASE_IMAGE"); env != "" {
			baseImage = env
		} else {
			baseImage = "spwn-world:latest"
		}
	}

	// Labels: identify the project + mark the image kind so
	// test cleanup can scope to built images without touching
	// world or architect containers.
	labels := map[string]string{
		"sh.spwn.kind":    "project-build",
		"sh.spwn.project": p.Manifest.Name,
		"sh.spwn.runtime": runtimeName,
		"sh.spwn.world":   worldID,
	}
	if runID := os.Getenv("SPWN_TEST_LABEL"); runID != "" {
		labels["sh.spwn.test.run"] = runID
	}

	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	// Docker client. Routed through backend.NewDockerClient so
	// `spwn build` picks up the same per-user socket discovery as
	// `spwn up` (OrbStack without admin rights, per-user Docker
	// Desktop, rootless Podman, etc.). stdout stays clean for --json;
	// build logs go to stderr further down.
	ctx := context.Background()
	dockerCli, err := backend.NewDockerClient(ctx)
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer dockerCli.Close()
	result, err := compile.BuildFromBase(ctx, dockerCli, compile.BuildFromBaseRequest{
		BaseImage:       baseImage,
		Tree:            tree,
		TreeDestination: "/world",
		Tag:             tag,
		Labels:          labels,
		NoCache:         buildNoCache,
		LogWriter:       errOut,
	})
	if err != nil {
		return fmt.Errorf("build image: %w", err)
	}

	treeFiles := len(tree.Paths())

	if buildJSON {
		report := buildReport{
			Runtime:   runtimeName,
			World:     worldID,
			TreeFiles: treeFiles,
			TreeOnly:  false,
			Tag:       result.Tag,
			ImageID:   result.ImageID,
			BaseImage: baseImage,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Fprintln(errOut)
	fmt.Fprintf(errOut, "  %s  %s\n", ui.Green("✓"), ui.Strong("Built image"))
	fmt.Fprintf(errOut, "     %s\n", ui.Faint(result.Tag))
	fmt.Fprintln(errOut)
	fmt.Fprintf(errOut, "  %d file(s) compiled from transpile tree, runtime=%s\n",
		treeFiles, runtimeName)
	if result.ImageID != "" {
		fmt.Fprintf(errOut, "  %s %s\n", ui.Faint("image id:"), ui.Faint(result.ImageID))
	}
	fmt.Fprintln(errOut)
	return nil
}

// ── shared helpers (formerly in transpile.go) ──────────────────────

// filterTreeByAgent returns a new Tree containing only entries that
// belong to the named agent (prefix "agents/<name>/"), along with the
// sorted set of agent names observed in the input tree — useful for
// error messages when the filter drops everything.
func filterTreeByAgent(t *transpile.Tree, name string) (*transpile.Tree, []string) {
	out := transpile.New()
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
func listTreeAgents(t *transpile.Tree) []string {
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
// build has a non-empty AGENTS.md on disk. Empty prompts would
// otherwise produce a silently-templated CLAUDE.md with no system
// instructions — worse than a loud error.
func requireAgentPrompts(src *source.ProjectSource, input transpile.Input) error {
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
// guard is cheap insurance against a bogus --output value wiping
// more than the compile tree.
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
		return fmt.Errorf("--output %s is not a directory", clean)
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
