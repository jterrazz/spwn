package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/compile"
	_ "spwn.sh/packages/compile/runtimes/claudecode" // register the claude-code runtime
	"spwn.sh/packages/compile/source"
	"spwn.sh/packages/image"
	"spwn.sh/packages/project"
)

func init() {
	buildCmd.Flags().StringVar(&buildRuntime, "runtime", "", "Target runtime. Defaults to the runtime declared in agent.yaml (fallback: claude-code)")
	buildCmd.Flags().StringVar(&buildWorld, "world", "", "World from spwn.yaml to build (required for multi-world projects)")
	buildCmd.Flags().StringVar(&buildTag, "tag", "", "Image tag (default: spwn-<project>:latest)")
	buildCmd.Flags().StringVar(&buildBase, "base", "", "Base image to derive from (default: $SPWN_BASE_IMAGE, else spwn-world:latest)")
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Disable Docker build cache")
	buildCmd.Flags().BoolVar(&buildJSON, "json", false, "Emit a machine-readable build report on stdout")
	rootCmd.AddCommand(buildCmd)
}

var (
	buildRuntime string
	buildWorld   string
	buildTag     string
	buildBase    string
	buildNoCache bool
	buildJSON    bool
)

// buildReport is the CLI-owned JSON schema for `spwn build --json`.
type buildReport struct {
	Runtime   string `json:"runtime"`
	Tag       string `json:"tag"`
	ImageID   string `json:"imageId"`
	BaseImage string `json:"baseImage"`
	TreeFiles int    `json:"treeFiles"`
	World     string `json:"world"`
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile the project and bake it into a Docker image",
	Long: `Compile the project with the target runtime (default: claude-code)
and bake the result into a derived Docker image.

The image is FROM spwn-world:latest by default, with the compiled
tree COPY'd to /world/. The resulting image carries the project's
name and the runtime name as Docker labels, so it's push-ready and
reproducible.

Use 'spwn compile' for the compile step alone (no Docker required).
Use 'spwn up' to spawn a world from the current project. Use 'spwn
check --deep' to run the compile dry-run as part of validation.

Examples:
  spwn build                                  # default tag: spwn-<project>:latest
  spwn build --tag spwn-myproj:v1
  spwn build --base spwn-world:2.1
  spwn build --runtime claude-code
  spwn build --world <name>                   # multi-world projects
  spwn build --no-cache
  spwn build --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Validate before touching Docker — same rules as `spwn
		// check`. This keeps bad manifests from turning into
		// confusing docker build errors downstream.
		issues := project.Validate(p, project.ValidateOpts{
			BuiltinTools:      catalogToolNames(),
			SupportedRuntimes: supportedRuntimes(),
		})
		if project.HasErrors(issues) {
			return fmt.Errorf("project has validation errors — run `spwn check` to see them")
		}

		src, err := source.Load(p.Root)
		if err != nil {
			return fmt.Errorf("load project source: %w", err)
		}

		// Resolve runtime: --runtime override > agent declaration >
		// claude-code fallback. The agent-declared path lands in a
		// follow-up commit; for now, only handle the override +
		// fallback.
		runtimeName, err := source.ResolveRuntime(src, buildRuntime)
		if err != nil {
			return err
		}

		input, err := source.ToCompileInput(src, buildWorld)
		if err != nil {
			return err
		}

		tree, err := compile.Compile(runtimeName, input)
		if err != nil {
			if strings.Contains(err.Error(), "unknown runtime") {
				return fmt.Errorf(
					"%v\n\nKnown runtimes: claude-code", err)
			}
			return fmt.Errorf("compile: %w", err)
		}

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
			"sh.spwn.world":   input.WorldID,
		}
		if runID := os.Getenv("SPWN_TEST_LABEL"); runID != "" {
			labels["sh.spwn.test.run"] = runID
		}

		out := cmd.OutOrStdout()
		errOut := cmd.ErrOrStderr()

		// Docker client. We stream build output to stderr so stdout
		// stays clean for --json.
		dockerCli, err := client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			return fmt.Errorf("docker client: %w", err)
		}
		defer dockerCli.Close()

		ctx := context.Background()
		result, err := image.BuildFromBase(ctx, dockerCli, image.BuildFromBaseRequest{
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
				Tag:       result.Tag,
				ImageID:   result.ImageID,
				BaseImage: baseImage,
				TreeFiles: treeFiles,
				World:     input.WorldID,
			}
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		}

		fmt.Fprintln(errOut)
		fmt.Fprintf(errOut, "  %s  %s\n", ui.Green("✓"), ui.Strong("Built image"))
		fmt.Fprintf(errOut, "     %s\n", ui.Faint(result.Tag))
		fmt.Fprintln(errOut)
		fmt.Fprintf(errOut, "  %d file(s) baked from compile tree, runtime=%s\n",
			treeFiles, runtimeName)
		if result.ImageID != "" {
			fmt.Fprintf(errOut, "  %s %s\n", ui.Faint("image id:"), ui.Faint(result.ImageID))
		}
		fmt.Fprintln(errOut)
		return nil
	},
}
