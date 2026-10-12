package dockerfile

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"spwn.sh/packages/dependency/tool"
)

func encodePolicyJSON(p *Policy) string {
	type wire struct {
		Allow []string `json:"allow,omitempty"`
		Deny  []string `json:"deny,omitempty"`
	}
	b, _ := json.Marshal(wire{Allow: p.Allow, Deny: p.Deny})
	return string(b)
}

func escapeSingleQuoteForShell(s string) string {
	return strings.ReplaceAll(s, "'", `'"'"'`)
}

// ToolInput is the data the generator needs from each tool.
type ToolInput struct {
	Name     string
	Packages tool.Packages
	Commands []string
	Env      map[string]string
	Files    map[string][]byte

	// Policy, when set, materializes a per-agent allow/deny filter
	// at /etc/spwn/policy/<short>.json. Catalog-tool wrappers (the
	// scripts emitted by Commands) read this file at runtime to
	// reject denied methods. Short is the tool's slug, e.g. "x" for
	// "spwn:x" — used both as the filename and as the JSON object's
	// implicit subject.
	Policy *Policy
}

// Policy is the on-image shape of an agent's allow/deny filter for
// one tool. Empty Allow + empty Deny means no policy file is
// written (caller normalizes).
type Policy struct {
	Short string
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// GenerateOpts configures Dockerfile generation.
type GenerateOpts struct {
	// SkipFooter disables the standard USER/WORKDIR/ENTRYPOINT footer.
	// Use this for non-standard images (e.g., architect) that define their own.
	SkipFooter bool

	// User is the non-root user in the image. Defaults to "spwn".
	// Used for the chown + USER directives in the footer.
	User string

	// Home is the user's home directory. Defaults to "/home/<User>".
	// Used for the WORKDIR directive in the footer.
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

func writeRunCommand(sb *strings.Builder, cmd string) {
	trimmed := strings.TrimRight(cmd, "\n")
	if !strings.ContainsRune(trimmed, '\n') {
		fmt.Fprintf(sb, "RUN %s\n", trimmed)
		return
	}
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

	sb.Write(baseDockerfile)
	sb.WriteString("\n")

	if imageVersion != "" {
		fmt.Fprintf(&sb, "LABEL sh.spwn.image-version=%q\n\n", imageVersion)
	}

	var allAptPackages []string
	seenApt := make(map[string]bool)
	for _, t := range tools {
		for _, pkg := range t.Packages.Apt {
			if !seenApt[pkg] {
				seenApt[pkg] = true
				allAptPackages = append(allAptPackages, pkg)
			}
		}
	}

	if len(allAptPackages) > 0 {
		sort.Strings(allAptPackages)
		sb.WriteString("# Packages (merged from all tools)\n")
		sb.WriteString("RUN apt-get update && apt-get install -y \\\n")
		for _, pkg := range allAptPackages {
			fmt.Fprintf(&sb, "    %s \\\n", pkg)
		}
		sb.WriteString("    && rm -rf /var/lib/apt/lists/*\n\n")
	}

	for _, t := range tools {
		hasContent := len(t.Commands) > 0 || len(t.Env) > 0 || len(t.Files) > 0
		if !hasContent {
			continue
		}

		fmt.Fprintf(&sb, "# %s\n", t.Name)

		envKeys := make([]string, 0, len(t.Env))
		for k := range t.Env {
			envKeys = append(envKeys, k)
		}
		sort.Strings(envKeys)
		for _, k := range envKeys {
			fmt.Fprintf(&sb, "ENV %s=%s\n", k, t.Env[k])
		}

		if len(t.Files) > 0 {
			paths := make([]string, 0, len(t.Files))
			for p := range t.Files {
				paths = append(paths, p)
			}
			sort.Strings(paths)
			for _, p := range paths {
				contextPath := fmt.Sprintf("tools/%s%s", t.Name, p)
				fmt.Fprintf(&sb, "COPY %s %s\n", contextPath, p)
			}
		}

		for _, cmd := range t.Commands {
			writeRunCommand(&sb, cmd)
		}

		if t.Policy != nil && (len(t.Policy.Allow) > 0 || len(t.Policy.Deny) > 0) && t.Policy.Short != "" {
			policyJSON := encodePolicyJSON(t.Policy)
			fmt.Fprintf(&sb,
				"RUN mkdir -p /etc/spwn/policy && printf '%%s' '%s' > /etc/spwn/policy/%s.json\n",
				escapeSingleQuoteForShell(policyJSON),
				t.Policy.Short,
			)
		}

		sb.WriteString("\n")
	}

	if !opt.SkipFooter {
		user := opt.user()
		home := opt.home()

		sb.WriteString("# Final setup\n")
		fmt.Fprintf(&sb, "RUN chown -R %s:%s %s\n", user, user, home)
		fmt.Fprintf(&sb, "USER %s\n", user)
		fmt.Fprintf(&sb, "WORKDIR %s\n", home)

		sb.WriteString("ENTRYPOINT [\"sleep\", \"infinity\"]\n")
	}

	return []byte(sb.String())
}
