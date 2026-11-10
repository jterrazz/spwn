package settings

import "strings"

// invariant: the ONLY home of the old product's literals: the block the base's
// install script wrote into ~/.zshrc, recognized here so `station init` can
// replace it. The rename greps exclude this file by name.
const (
	legacyMarker = "# studio-workstation"
	legacyVar    = "WORKSTATION_REPO_PATH"
	legacyLoader = "dotfiles/applications/zsh/zshrc.sh"
)

func legacy(line string) bool {
	return strings.Contains(line, legacyVar) || strings.Contains(line, legacyLoader)
}

func withoutLegacy(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == legacyMarker {
			for i+1 < len(lines) && legacy(lines[i+1]) {
				i++
			}
			continue
		}
		if legacy(lines[i]) {
			continue
		}
		out = append(out, lines[i])
	}
	return out
}
