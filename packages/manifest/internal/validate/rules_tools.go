package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ruleWorldToolsExist checks that every tool declared in the world
// config is either:
//
//   - a built-in tool pack from the imagebuilder catalog (passed in
//     via Input.BuiltinTools by the caller), or
//   - a directory under ./spwn/tools/<name>/ (local tool pack).
//
// When BuiltinTools is nil the caller didn't inject a catalog, so we
// fall back to a simple "@spwn/*" prefix check — enough to catch
// obvious typos without false positives.
func ruleWorldToolsExist(in Input) []Issue {
	if !in.WorldExists {
		return nil
	}
	worldTools, err := loadWorldTools(in.WorldPath)
	if err != nil {
		return nil // parse errors handled elsewhere
	}
	builtin := make(map[string]struct{}, len(in.BuiltinTools))
	for _, t := range in.BuiltinTools {
		builtin[t] = struct{}{}
	}

	var out []Issue
	for _, tool := range worldTools {
		if resolveTool(in.Root, tool, builtin, in.BuiltinTools != nil) {
			continue
		}
		out = append(out, Issue{
			Level:   LevelError,
			Path:    relPath(in.Root, in.WorldPath) + "#tools",
			Message: fmt.Sprintf("tool %q does not exist", tool),
			Hint:    suggestTool(tool, in.BuiltinTools),
		})
	}
	return out
}

// resolveTool returns true when tool is either in the catalog, a
// ./spwn/tools/<name>/ directory, or (fallback) looks like a valid
// @spwn/* name.
func resolveTool(root, tool string, builtin map[string]struct{}, haveCatalog bool) bool {
	// Built-in lookup: authoritative when the caller provided a catalog.
	if haveCatalog {
		if _, ok := builtin[tool]; ok {
			return true
		}
	} else {
		// Best-effort without a catalog: accept anything that looks
		// like a scoped tool name.
		if strings.HasPrefix(tool, "@spwn/") {
			return true
		}
	}

	// Local tool pack: ./spwn/tools/<name>/ where <name> is the tool
	// name with any leading @scope/ stripped.
	localName := tool
	if idx := strings.Index(tool, "/"); strings.HasPrefix(tool, "@") && idx > 0 {
		localName = tool[idx+1:]
	}
	localPath := filepath.Join(root, "spwn", "tools", localName)
	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		return true
	}
	return false
}

// suggestTool produces a short hint when a tool isn't found. Tries a
// quick Levenshtein-style match against the catalog for "did you mean"
// friendliness, falls back to a generic message otherwise.
func suggestTool(tool string, catalog []string) string {
	if len(catalog) == 0 {
		return "check the tool name, or add it as a local pack under ./spwn/tools/"
	}
	best := ""
	bestScore := len(tool) + 1
	for _, c := range catalog {
		if d := editDistance(tool, c); d < bestScore && d <= 3 {
			best = c
			bestScore = d
		}
	}
	if best != "" {
		return "did you mean " + best + "?"
	}
	return "available built-ins: " + strings.Join(catalog, ", ")
}

// editDistance is a small Levenshtein implementation. Good enough for
// "did you mean" on short tool names (typically < 30 chars).
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = min3(del, ins, sub)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
