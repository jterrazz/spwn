package catalog

import (
	"regexp"
	"strings"
)

const (
	bunfigStart = "# >>> station >>>"
	bunfigEnd   = "# <<< station <<<"
)

var ownInstallTable = regexp.MustCompile(`(?m)^\s*\[install(\.cache)?\]\s*$`)

func bunfigBlock(globalDir, globalBinDir, cacheDir string) string {
	return bunfigStart + "\n" +
		"[install]\n" +
		"globalDir = \"" + globalDir + "\"\n" +
		"globalBinDir = \"" + globalBinDir + "\"\n" +
		"\n" +
		"[install.cache]\n" +
		"dir = \"" + cacheDir + "\"\n" +
		bunfigEnd + "\n"
}

func UpsertBunfig(text, globalDir, globalBinDir, cacheDir string) (updated string, changed bool, conflict string) {
	block := bunfigBlock(globalDir, globalBinDir, cacheDir)
	before, rest, managed := strings.Cut(text, bunfigStart)
	after := ""
	if managed {
		_, after, _ = strings.Cut(rest, bunfigEnd+"\n")
	}
	if ownInstallTable.MatchString(before + after) {
		return text, false, "~/.bunfig.toml has its own [install]: set globalDir, globalBinDir and cache.dir by hand"
	}
	if managed {
		updated = before + block + after
		return updated, updated != text, ""
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text + block, true, ""
}
