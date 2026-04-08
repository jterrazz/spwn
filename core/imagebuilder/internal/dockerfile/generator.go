package dockerfile

import (
	"fmt"
	"sort"
	"strings"
)

// ToolInput is the data the generator needs from each tool.
type ToolInput struct {
	Name     string
	Kind     string
	Packages []string
	Commands []string
	Env      map[string]string
	Files    map[string][]byte
}

// Generate composes a final Dockerfile from a base Dockerfile and tool inputs.
// Tools must be in topological order (dependencies first).
// The base Dockerfile must NOT contain USER/WORKDIR/ENTRYPOINT — those are added
// as a footer after all tools install (tools need root access).
func Generate(baseDockerfile []byte, tools []ToolInput, imageVersion string) []byte {
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

	// Per-tool sections: ENV, FILES, RUN commands
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

	// Footer: fix ownership and switch to non-root user
	sb.WriteString("# Final setup\n")
	sb.WriteString("RUN chown -R spwn:spwn /home/spwn\n")
	sb.WriteString("USER spwn\n")
	sb.WriteString("WORKDIR /home/spwn\n")
	sb.WriteString("VOLUME [\"/workspace\", \"/mind\", \"/universe\", \"/world\"]\n")
	sb.WriteString("ENTRYPOINT [\"sleep\", \"infinity\"]\n")

	return []byte(sb.String())
}
