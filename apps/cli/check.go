package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/compile"
	"spwn.sh/packages/compile/source"
	"spwn.sh/packages/project"
)

func init() {
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "Exit non-zero on warnings, not just errors")
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Emit results as structured JSON on stdout")
	checkCmd.Flags().BoolVar(&checkDeep, "deep", false, "Also run a compile dry-run and report renderer errors")
	rootCmd.AddCommand(checkCmd)
}

var (
	checkStrict bool
	checkJSON   bool
	checkDeep   bool
)

// checkReport is the CLI-owned JSON schema for `spwn check`. It's
// intentionally decoupled from the internal validate.Issue type so the
// JSON contract can evolve independently of the rule engine internals.
type checkReport struct {
	Valid        bool          `json:"valid"`
	ManifestPath string        `json:"manifestPath"`
	Summary      checkSummary  `json:"summary"`
	Issues       []checkIssue  `json:"issues"`
}

type checkSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

type checkIssue struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	// Source identifies which pass produced the issue. "manifest"
	// (default, from the project.Validate rule engine) or "compile"
	// (only emitted under --deep).
	Source string `json:"source,omitempty"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate the project tree against spwn.yaml",
	Long: `Walks up from the current directory looking for spwn.yaml, then runs
every validation rule against the project. Reports issues grouped by
severity. Exits non-zero when errors are found (or warnings, with
--strict).`,
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
			return fmt.Errorf("no spwn.yaml found in %s or any parent directory.\nRun `spwn init` to create one", cwd)
		}

		issues := project.Validate(p, project.ValidateOpts{
			BuiltinTools:      catalogToolNames(),
			BuiltinSkills:     catalogSkillNames(),
			SupportedRuntimes: supportedRuntimes(),
		})
		out := cmd.OutOrStdout()

		var compileIssues []compileIssue
		if checkDeep {
			compileIssues = runCompileDeepCheck(p.Root)
		}
		// Cross-check: an agent may declare a runtime the catalog
		// recognises (e.g. `@spwn/codex`) even when no compile
		// adapter is registered for it. Without this warning,
		// `spwn check` happily reports "valid" and the user only
		// discovers the gap when `spwn build` fails with
		// "unknown runtime". Surface it as a warning at every
		// severity (not just --deep) so the inconsistency is loud.
		compileIssues = append(compileIssues, crossCheckRuntimeAdapters(p.Root)...)

		errors := filter(issues, project.LevelError)
		warnings := filter(issues, project.LevelWarning)
		infos := filter(issues, project.LevelInfo)

		if checkJSON {
			report := buildCheckReport(p.ManifestPath, errors, warnings, infos, compileIssues)
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			if err := enc.Encode(report); err != nil {
				return fmt.Errorf("encode json: %w", err)
			}
			if len(errors) > 0 || hasCompileErrors(compileIssues) || (checkStrict && len(warnings) > 0) {
				os.Exit(1)
			}
			return nil
		}

		if len(issues) == 0 && len(compileIssues) == 0 {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "  %s  %s\n", ui.Green("✓"), ui.Strong("Project is valid"))
			fmt.Fprintf(out, "     %s\n", ui.Faint(p.ManifestPath))
			fmt.Fprintln(out)
			return nil
		}

		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %s\n", ui.Faint(p.ManifestPath))
		fmt.Fprintln(out)

		printGroup(out, "errors", errors, ui.Red)
		printGroup(out, "warnings", warnings, ui.Yellow)
		printGroup(out, "info", infos, ui.Faint)
		printCompileIssues(out, compileIssues)

		fmt.Fprintf(out, "  %d error(s), %d warning(s), %d info\n",
			len(errors)+countCompileErrors(compileIssues),
			len(warnings)+countCompileWarnings(compileIssues),
			len(infos))
		fmt.Fprintln(out)

		compileWarns := countCompileWarnings(compileIssues)
		if len(errors) > 0 || hasCompileErrors(compileIssues) ||
			(checkStrict && (len(warnings)+compileWarns) > 0) {
			os.Exit(1)
		}
		return nil
	},
}

func buildCheckReport(manifestPath string, errors, warnings, infos []project.Issue, compileIssues []compileIssue) checkReport {
	issues := make([]checkIssue, 0, len(errors)+len(warnings)+len(infos)+len(compileIssues))
	for _, group := range [][]project.Issue{errors, warnings, infos} {
		for _, i := range group {
			issues = append(issues, checkIssue{
				Level:   i.Level.String(),
				Path:    i.Path,
				Message: i.Message,
				Hint:    i.Hint,
				Source:  "manifest",
			})
		}
	}
	for _, ci := range compileIssues {
		issues = append(issues, checkIssue{
			Level:   ci.Level,
			Path:    ci.Path,
			Message: ci.Message,
			Hint:    ci.Hint,
			Source:  "compile",
		})
	}
	compileErrs := countCompileErrors(compileIssues)
	return checkReport{
		Valid:        len(errors) == 0 && compileErrs == 0,
		ManifestPath: manifestPath,
		Summary: checkSummary{
			Errors:   len(errors) + compileErrs,
			Warnings: len(warnings) + countCompileWarnings(compileIssues),
			Info:     len(infos),
		},
		Issues: issues,
	}
}

// compileIssue is one finding produced by the --deep compile pass.
// Kept parallel to project.Issue so the two can be merged in the
// report without forcing a dependency from packages/compile onto the
// project rule engine.
type compileIssue struct {
	Level   string // "error" | "warning"
	Path    string
	Message string
	Hint    string
}

// runCompileDeepCheck loads the project from disk and runs a compile
// dry-run. Errors become "error" issues; suspicious render output
// (agent with no files in the Tree) becomes "warning" issues.
func runCompileDeepCheck(projectRoot string) []compileIssue {
	src, err := source.Load(projectRoot)
	if err != nil {
		return []compileIssue{{
			Level:   "error",
			Path:    projectRoot,
			Message: fmt.Sprintf("load project source: %v", err),
			Hint:    "every file under spwn/ must be readable",
		}}
	}
	input, err := source.ToCompileInput(src, "")
	if err != nil {
		// Multi-world projects (or manifests with no world at all)
		// surface as a warning rather than an error — deep-check is
		// best-effort, not a world selector. `check` has no --world
		// flag of its own, so the hint points users at the compile
		// command.
		return []compileIssue{{
			Level:   "warning",
			Path:    "spwn.yaml#worlds",
			Message: fmt.Sprintf("compile pass skipped: %v", err),
			Hint:    "run `spwn build --tree-only --world <name>` against each world to validate it",
		}}
	}
	var issues []compileIssue

	// Every agent in the selected world must have an AGENTS.md on
	// disk — the manifest rule engine checks that spwn/agents/<name>/
	// exists, but not that the entrypoint prompt is there. Missing
	// AGENTS.md would silently render an empty CLAUDE.md, which is
	// worse than a loud error, so the deep check promotes it.
	byName := make(map[string]int, len(src.Agents))
	for i, a := range src.Agents {
		byName[a.Name] = i
	}
	for _, a := range input.Agents {
		idx, ok := byName[a.Name]
		if !ok {
			continue
		}
		if len(src.Agents[idx].AgentMD) == 0 {
			issues = append(issues, compileIssue{
				Level:   "error",
				Path:    "spwn/agents/" + a.Name + "/AGENTS.md",
				Message: "agent prompt is missing or empty",
				Hint:    "create spwn/agents/" + a.Name + "/AGENTS.md with the agent's system prompt",
			})
		}
	}

	tree, err := compile.Compile("claude-code", input)
	if err != nil {
		return append(issues, compileIssue{
			Level:   "error",
			Path:    "compile:claude-code",
			Message: err.Error(),
			Hint:    "runtime renderer rejected the project",
		})
	}
	// Warn about any roster agent that produced no files in the
	// Tree — usually means the agent directory is missing the
	// entrypoint or the renderer silently dropped it.
	for _, a := range input.Agents {
		prefix := "agents/" + a.Name + "/"
		found := false
		for _, p := range tree.Paths() {
			if len(p) >= len(prefix) && p[:len(prefix)] == prefix {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, compileIssue{
				Level:   "warning",
				Path:    "agents/" + a.Name,
				Message: "agent produced no files in the compiled Tree",
				Hint:    "check spwn/agents/" + a.Name + "/ for missing AGENTS.md or agent.yaml",
			})
		}
	}
	return issues
}

// crossCheckRuntimeAdapters inspects each agent.yaml's declared
// runtime.backend and warns when the value is catalog-recognised but
// has no compile adapter registered. This closes the gap where
// `spwn check` says "valid" and `spwn build` then fails with
// `unknown runtime`.
func crossCheckRuntimeAdapters(projectRoot string) []compileIssue {
	src, err := source.Load(projectRoot)
	if err != nil || src == nil {
		return nil
	}
	registered := map[string]struct{}{}
	for _, name := range compile.RegisteredRuntimes() {
		registered[name] = struct{}{}
	}
	var out []compileIssue
	for _, a := range src.Agents {
		raw := a.Config.Runtime.Backend
		if raw == "" {
			continue
		}
		// Map catalog refs like "@spwn/codex" -> "codex" before
		// looking them up. ResolveRuntime uses the same mapping.
		canonical := raw
		if strings.HasPrefix(raw, "@spwn/") {
			canonical = strings.TrimPrefix(raw, "@spwn/")
		}
		if _, ok := registered[canonical]; ok {
			continue
		}
		known := strings.Join(compile.RegisteredRuntimes(), ", ")
		out = append(out, compileIssue{
			Level:   "warning",
			Path:    "spwn/agents/" + a.Name + "/agent.yaml#runtime.backend",
			Message: fmt.Sprintf("runtime %q has no compile adapter registered", raw),
			Hint:    "available compile runtimes: " + known,
		})
	}
	return out
}

func hasCompileErrors(ci []compileIssue) bool {
	for _, c := range ci {
		if c.Level == "error" {
			return true
		}
	}
	return false
}

func countCompileErrors(ci []compileIssue) int {
	n := 0
	for _, c := range ci {
		if c.Level == "error" {
			n++
		}
	}
	return n
}

func countCompileWarnings(ci []compileIssue) int {
	n := 0
	for _, c := range ci {
		if c.Level == "warning" {
			n++
		}
	}
	return n
}

func printCompileIssues(out interface{ Write([]byte) (int, error) }, ci []compileIssue) {
	if len(ci) == 0 {
		return
	}
	// Header color reflects the max severity present: red when any
	// issue is an error, yellow for warning-only batches. Previously
	// always red — misleading for deep-check "skipped" warnings.
	headerColor := ui.Yellow
	if hasCompileErrors(ci) {
		headerColor = ui.Red
	}
	fmt.Fprintf(out, "  %s\n", headerColor("compile"))
	for _, c := range ci {
		color := ui.Red
		if c.Level == "warning" {
			color = ui.Yellow
		}
		fmt.Fprintf(out, "    %s  %s\n", color("•"), c.Message)
		fmt.Fprintf(out, "       %s\n", ui.Faint(c.Path))
		if c.Hint != "" {
			fmt.Fprintf(out, "       %s %s\n", ui.Faint("→"), ui.Faint(c.Hint))
		}
	}
	fmt.Fprintln(out)
}

func filter(issues []project.Issue, level project.Level) []project.Issue {
	var out []project.Issue
	for _, i := range issues {
		if i.Level == level {
			out = append(out, i)
		}
	}
	return out
}

func printGroup(out interface{ Write([]byte) (int, error) }, label string, issues []project.Issue, color func(string) string) {
	if len(issues) == 0 {
		return
	}
	fmt.Fprintf(out, "  %s\n", color(label))
	for _, i := range issues {
		fmt.Fprintf(out, "    %s  %s\n", color("•"), i.Message)
		fmt.Fprintf(out, "       %s\n", ui.Faint(i.Path))
		if i.Hint != "" {
			fmt.Fprintf(out, "       %s %s\n", ui.Faint("→"), ui.Faint(i.Hint))
		}
	}
	fmt.Fprintln(out)
}
