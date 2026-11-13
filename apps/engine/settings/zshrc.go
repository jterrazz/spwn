package settings

import (
	"encoding/json"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const (
	blockStart = "# >>> station >>>"
	blockEnd   = "# <<< station <<<"
)

// invariant: ~/.local/bin is on PATH before the eval runs, and the `command -v`
// guard keeps a deleted binary from breaking every new shell.
const block = blockStart + "\n" +
	`export PATH="$HOME/.local/bin:$PATH"` + "\n" +
	`command -v station >/dev/null 2>&1 && eval "$(station shell zsh)"` + "\n" +
	blockEnd + "\n"

func HasBlock(zshrc string) bool {
	return strings.Count(zshrc, block) == 1
}

func UpsertBlock(zshrc string) (updated string, changed bool) {
	lines := strings.Split(zshrc, "\n")
	kept := withoutLegacy(lines)
	if HasBlock(zshrc) && len(kept) == len(lines) && strings.Count(zshrc, blockStart) == 1 && mainFirst(lines) {
		return zshrc, false
	}
	var out []string
	inBlock := false
	for _, line := range kept {
		switch {
		case strings.TrimSpace(line) == blockStart:
			inBlock = true
		case strings.TrimSpace(line) == blockEnd:
			inBlock = false
		case !inBlock:
			out = append(out, line)
		}
	}
	if spans := spansOf(out); len(spans) > 0 {
		head := strings.TrimRight(strings.Join(out[:spans[0].start], "\n"), "\n")
		if head != "" {
			head += "\n\n"
		}
		updated = head + block + strings.Join(out[spans[0].start:], "\n")
		return updated, updated != zshrc
	}
	text := strings.TrimRight(strings.Join(out, "\n"), "\n")
	if text != "" {
		text += "\n\n"
	}
	updated = text + block
	return updated, updated != zshrc
}

// value: where station's main block sits in the lines, its marker lines
// included, when both are there.
func mainSpan(lines []string) (start, end int, ok bool) {
	start = -1
	for i, line := range lines {
		switch t := strings.TrimSpace(line); {
		case t == blockStart && start < 0:
			start = i
		case t == blockEnd && start >= 0:
			return start, i, true
		}
	}
	return 0, 0, false
}

// invariant: station's main block comes before every package block.
func mainFirst(lines []string) bool {
	spans := spansOf(lines)
	if len(spans) == 0 {
		return true
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == blockStart {
			return i < spans[0].start
		}
	}
	return true
}

func ClaudeTokenLine(exportJSON []byte) (string, bool) {
	var values map[string]string
	if json.Unmarshal(exportJSON, &values) != nil {
		return "", false
	}
	token := values[catalog.ClaudeTokenName]
	if token == "" {
		return "", false
	}
	return "export CLAUDE_CODE_OAUTH_TOKEN='" + strings.ReplaceAll(token, "'", `'\''`) + "'", true
}
