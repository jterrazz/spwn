package dockerfile

import (
	"fmt"
	"sort"
	"strings"
)

// ToolInput is the data the generator needs from each tool.
type ToolInput struct {
	Name         string
	Kind         string
	Packages     []string
	Commands     []string
	UserCommands []string // Commands that run after USER switch (templates: {{.Home}}, {{.User}})
	Env          map[string]string
	Files        map[string][]byte
	HasSkills    bool // If true, generator adds COPY for skills directory
}

// GenerateOpts configures Dockerfile generation.
type GenerateOpts struct {
	// SkipFooter disables the standard USER/WORKDIR/VOLUME/ENTRYPOINT footer.
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
		for _, pkg := range t.Packages {
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

		sb.WriteString(fmt.Sprintf("# %s (%s)\n", t.Name, t.Kind))

		for k, v := range t.Env {
			sb.WriteString(fmt.Sprintf("ENV %s=%s\n", k, v))
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
			sb.WriteString(fmt.Sprintf("RUN %s\n", cmd))
		}

		sb.WriteString("\n")
	}

	// Copy skills into the image (if any tool has skills)
	// Skip when SkipFooter is set - the caller manages their own COPY directives.
	if !opt.SkipFooter {
		hasAnySkills := false
		for _, t := range tools {
			if t.HasSkills {
				hasAnySkills = true
				break
			}
		}
		if hasAnySkills {
			sb.WriteString("# Skills (copied from build context)\n")
			sb.WriteString("COPY skills/ /\n\n")
		}
	}

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
			sb.WriteString(fmt.Sprintf("RUN %s\n", cmd))
		}

		sb.WriteString("VOLUME [\"/work\", \"/agents\", \"/world\"]\n")
		sb.WriteString("ENTRYPOINT [\"sleep\", \"infinity\"]\n")
	}

	return []byte(sb.String())
}
