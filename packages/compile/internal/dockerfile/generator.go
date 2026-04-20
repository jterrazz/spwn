package dockerfile

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

// ToolInput is the data the generator needs from each tool.
type ToolInput struct {
	Name         string
	AptPackages  []string
	Commands     []string
	UserCommands []string // Commands that run after USER switch (templates: {{.Home}}, {{.User}})
	Env          map[string]string
	Files        map[string][]byte
}

// GenerateOpts configures Dockerfile generation.
type GenerateOpts struct {
	// SkipFooter disables the standard USER/WORKDIR/ENTRYPOINT footer.
	// Use this for non-standard images (e.g., architect) that define their own.
	SkipFooter bool

	// User is the non-root user in the image. Defaults to "spwn".
	// Used to template {{.User}} in UserCommands and for chown/USER directives.
	User string

	// Home is the user's home directory. Defaults to "/home/<User>".
	// Used to template {{.Home}} in UserCommands.
	Home string
}

func (o GenerateOpts) user() string {
	if o.User != "" {
		return o.User
	}
	return "spwn"
}

func (o GenerateOpts) home() string {
	if o.Home != "" {
		return o.Home
	}
	return "/home/" + o.user()
}

// templateUserCmd replaces {{.Home}} and {{.User}} in a command string.
func (o GenerateOpts) templateUserCmd(cmd string) string {
	cmd = strings.ReplaceAll(cmd, "{{.Home}}", o.home())
	cmd = strings.ReplaceAll(cmd, "{{.User}}", o.user())
	return cmd
}

// writeRunCommand emits a `RUN` directive for cmd. Single-line
// commands land as plain `RUN <cmd>`. Multi-line commands — typically
// tool.yaml entries authored as a YAML `|` block scalar that wants to
// `cat > /bin/foo <<HEREDOC` — are wrapped so Docker treats the
// whole block as one shell invocation.
//
// Plain `RUN` only accepts one shell-command-line; a naive
// fmt.Sprintf("RUN %s\n", multiLine) emits the first line as the
// command and the rest as subsequent unrelated Dockerfile lines,
// which parses but does nothing useful — shell heredocs end up as
// empty files. We pipe the whole thing through `bash -c`, which
// gives the author the shell semantics they expect (including
// heredocs, `set -e`, subshells, pipes) without them having to
// escape anything.
func writeRunCommand(sb *strings.Builder, cmd string) {
	trimmed := strings.TrimRight(cmd, "\n")
	if !strings.ContainsRune(trimmed, '\n') {
		sb.WriteString(fmt.Sprintf("RUN %s\n", trimmed))
		return
	}
	// Multi-line cmd — tool.yaml authored as a YAML `|` block scalar,
	// typically wrapping a `cat > /bin/foo <<HEREDOC`. Plain `RUN
	// <multi-line>` makes Docker split on newlines and emit only the
	// first line, dropping the heredoc body + EOF marker and producing
	// empty files. We encode the whole command as base64 and decode-
	// execute at build time — zero escaping concerns, preserves exact
	// bytes, works with plain Dockerfile syntax (no BuildKit heredoc
	// dependency).
	encoded := base64.StdEncoding.EncodeToString([]byte(trimmed))
	sb.WriteString("RUN echo " + encoded + " | base64 -d | bash -e\n")
}

// Generate composes a final Dockerfile from a base Dockerfile and tool inputs.
// Tools must be in topological order (dependencies first).
func Generate(baseDockerfile []byte, tools []ToolInput, imageVersion string, opts ...GenerateOpts) []byte {
	var opt GenerateOpts
	if len(opts) > 0 {
		opt = opts[0]
	}
	var sb strings.Builder

	// Write base Dockerfile (OS + user creation, still running as root)
	sb.Write(baseDockerfile)
	sb.WriteString("\n")

	// Version label
	if imageVersion != "" {
		sb.WriteString(fmt.Sprintf("LABEL sh.spwn.image-version=%q\n\n", imageVersion))
	}

	// Collect all apt packages across tools (deduplicated)
	var allPackages []string
	seen := make(map[string]bool)
	for _, t := range tools {
		for _, pkg := range t.AptPackages {
			if !seen[pkg] {
				seen[pkg] = true
				allPackages = append(allPackages, pkg)
			}
		}
	}

	// Single apt-get install for all packages
	if len(allPackages) > 0 {
		sort.Strings(allPackages)
		sb.WriteString("# Packages (merged from all tools)\n")
		sb.WriteString("RUN apt-get update && apt-get install -y \\\n")
		for _, pkg := range allPackages {
			sb.WriteString(fmt.Sprintf("    %s \\\n", pkg))
		}
		sb.WriteString("    && rm -rf /var/lib/apt/lists/*\n\n")
	}

	// Per-tool sections: ENV, FILES, RUN commands (all as root)
	for _, t := range tools {
		hasContent := len(t.Commands) > 0 || len(t.Env) > 0 || len(t.Files) > 0
		if !hasContent {
			continue
		}

		sb.WriteString(fmt.Sprintf("# %s\n", t.Name))

		// Sort env keys for deterministic output (Go map iteration
		// order is randomized; content-addressed image hashing
		// requires stable bytes).
		envKeys := make([]string, 0, len(t.Env))
		for k := range t.Env {
			envKeys = append(envKeys, k)
		}
		sort.Strings(envKeys)
		for _, k := range envKeys {
			sb.WriteString(fmt.Sprintf("ENV %s=%s\n", k, t.Env[k]))
		}

		if len(t.Files) > 0 {
			paths := make([]string, 0, len(t.Files))
			for p := range t.Files {
				paths = append(paths, p)
			}
			sort.Strings(paths)
			for _, p := range paths {
				contextPath := fmt.Sprintf("tools/%s%s", t.Name, p)
				sb.WriteString(fmt.Sprintf("COPY %s %s\n", contextPath, p))
			}
		}

		for _, cmd := range t.Commands {
			writeRunCommand(&sb, cmd)
		}

		sb.WriteString("\n")
	}

	// Skills no longer live in the image — the transpile layer writes
	// them into each agent's `.claude/skills/` / `.agents/skills/` at
	// spawn time via docker-cp. No COPY directive needed here.

	// Collect UserCommands across all tools (run after USER switch)
	var allUserCmds []string
	for _, t := range tools {
		for _, cmd := range t.UserCommands {
			allUserCmds = append(allUserCmds, opt.templateUserCmd(cmd))
		}
	}

	if !opt.SkipFooter {
		user := opt.user()
		home := opt.home()

		// Fix ownership before switching user
		sb.WriteString("# Final setup\n")
		sb.WriteString(fmt.Sprintf("RUN chown -R %s:%s %s\n", user, user, home))
		sb.WriteString(fmt.Sprintf("USER %s\n", user))
		sb.WriteString(fmt.Sprintf("WORKDIR %s\n", home))

		// Run user-level setup commands (config files, etc.)
		for _, cmd := range allUserCmds {
			writeRunCommand(&sb, cmd)
		}

		// No VOLUME declaration. /world and /workspaces/<name> are
		// bind-mounted by the spawner at container creation time;
		// pre-declaring them as image volumes makes Docker auto-create
		// anonymous volumes per container that `docker rm` never
		// cleans up (leading to 10k+ leaked volumes on heavy e2e runs).
		sb.WriteString("ENTRYPOINT [\"sleep\", \"infinity\"]\n")
	}

	return []byte(sb.String())
}
