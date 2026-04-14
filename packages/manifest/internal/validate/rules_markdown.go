package validate

import (
	"fmt"
	"path/filepath"
	"strings"

	"spwn.sh/packages/manifest/internal/resolve"
)

// ruleMarkdownImports walks each agent's CLAUDE.md and follows every
// @-import to make sure the target files exist. It also flags import
// cycles, which are technically harmless at runtime but a sign of
// copy-paste mistakes.
func ruleMarkdownImports(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for i := range in.Manifest.Agents {
		if i >= len(in.AgentPaths) || i >= len(in.AgentExists) || !in.AgentExists[i] {
			continue
		}
		agentDir := in.AgentPaths[i]
		claudePath := filepath.Join(agentDir, "CLAUDE.md")

		result, err := resolve.Walk(agentDir, claudePath)
		if err != nil {
			// Unreadable CLAUDE.md is already caught by
			// ruleAgentDirs; don't double-report.
			continue
		}

		for _, ref := range result.Missing {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    fmt.Sprintf("%s @%s", relPath(in.Root, ref.Source), ref.Target),
				Message: "broken @-import: " + ref.Target,
				Hint:    "create " + relPath(in.Root, ref.ResolvedPath) + " or remove the reference",
			})
		}

		for _, cycle := range result.Cycles {
			rel := make([]string, len(cycle))
			for j, p := range cycle {
				rel[j] = relPath(in.Root, p)
			}
			out = append(out, Issue{
				Level:   LevelWarning,
				Path:    relPath(in.Root, claudePath),
				Message: "import cycle: " + strings.Join(rel, " → "),
				Hint:    "break the cycle by removing one of the @-imports",
			})
		}
	}
	return out
}
