package dockerfile

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"spwn.sh/packages/dependency/tool"
)

// encodePolicyJSON serializes a Policy to its on-disk shape. Keeps
// only the allow/deny lists — short is encoded in the filename, not
// the body.
func encodePolicyJSON(p *Policy) string {
	type wire struct {
		Allow []string `json:"allow,omitempty"`
		Deny  []string `json:"deny,omitempty"`
	}
	b, _ := json.Marshal(wire{Allow: p.Allow, Deny: p.Deny})
	return string(b)
}

// escapeSingleQuoteForShell turns ' into '"'"' so the value can be
// embedded inside a single-quoted shell argument without breaking
// out of it. Shell-safe; no risk of arbitrary execution from policy
// content because we control the entire string and JSON-escape it
// first.
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
	Short string   // slug used in the filename (e.g. "x")
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

	// Collect apt packages across tools (deduplicated). New
	// managers (apk/brew/...) add their own collect+emit block
	// mirroring this one — each manager has its own flag shape and
	// cache-cleanup pattern, so they don't trivially share code.
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

	// Single apt-get install for all apt packages
	if len(allAptPackages) > 0 {
		sort.Strings(allAptPackages)
		sb.WriteString("# Packages (merged from all tools)\n")
		sb.WriteString("RUN apt-get update && apt-get install -y \\\n")
		for _, pkg := range allAptPackages {
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

		// Per-agent method allow/deny — materialized as a JSON file
		// the tool's wrapper reads at runtime. Catalog tools emit a
		// `spwn-policy-check <short> <method>` step in their wrappers
		// that consults this file.
		if t.Policy != nil && (len(t.Policy.Allow) > 0 || len(t.Policy.Deny) > 0) && t.Policy.Short != "" {
			policyJSON := encodePolicyJSON(t.Policy)
			sb.WriteString(fmt.Sprintf(
				"RUN mkdir -p /etc/spwn/policy && printf '%%s' '%s' > /etc/spwn/policy/%s.json\n",
				escapeSingleQuoteForShell(policyJSON),
				t.Policy.Short,
			))
		}

		sb.WriteString("\n")
	}

	// Skills no longer live in the image — the transpile layer writes
	// them into each agent's `.claude/skills/` / `.agents/skills/` at
	// spawn time via docker-cp. No COPY directive needed here.

	if !opt.SkipFooter {
		user := opt.user()
		home := opt.home()

		// Fix ownership before switching user
		sb.WriteString("# Final setup\n")
		sb.WriteString(fmt.Sprintf("RUN chown -R %s:%s %s\n", user, user, home))
		sb.WriteString(fmt.Sprintf("USER %s\n", user))
		sb.WriteString(fmt.Sprintf("WORKDIR %s\n", home))

		// No VOLUME declaration. /world and /workspaces/<name> are
		// bind-mounted by the spawner at container creation time;
		// pre-declaring them as image volumes makes Docker auto-create
		// anonymous volumes per container that `docker rm` never
		// cleans up (leading to 10k+ leaked volumes on heavy e2e runs).
		sb.WriteString("ENTRYPOINT [\"sleep\", \"infinity\"]\n")
	}

	return []byte(sb.String())
}
