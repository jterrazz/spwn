package calls

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/disk"
)

func UpsertTableKey(text, key, value string) (updated string, changed bool, handSet string) {
	if current, ok := disk.TableValue(text, key); ok {
		if current == value {
			return text, false, ""
		}
		return text, false, current
	}
	cut := strings.LastIndex(key, ".")
	header, line := "["+key[:cut]+"]", key[cut+1:]+" = "+value
	lines := strings.SplitAfter(text, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == header {
			if !strings.HasSuffix(l, "\n") {
				lines[i] += "\n"
			}
			return strings.Join(lines[:i+1], "") + line + "\n" + strings.Join(lines[i+1:], ""), true, ""
		}
	}
	switch {
	case text == "":
	case strings.HasSuffix(text, "\n"):
		text += "\n"
	default:
		text += "\n\n"
	}
	return text + header + "\n" + line + "\n", true, ""
}
