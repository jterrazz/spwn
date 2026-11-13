package disk

import (
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/setup"
)

type Expander struct {
	Home  string
	Roots setup.Roots
	Root  string
}

func (e Expander) Expand(pattern string) string {
	if pattern == "~" {
		return e.Home
	}
	if rest, ok := strings.CutPrefix(pattern, "~/"); ok {
		return path.Join(e.Home, rest)
	}
	if strings.HasPrefix(pattern, "/") {
		return path.Join("/", e.Root, pattern)
	}
	for prefix, root := range map[string]string{"{config}": e.Roots.Prefs, "{data}": e.Roots.Data, "{state}": e.Roots.State, "{cache}": e.Roots.Cache} {
		if pattern == prefix {
			return root
		}
		if rest, ok := strings.CutPrefix(pattern, prefix+"/"); ok {
			return path.Join(root, rest)
		}
	}
	return pattern
}

func (e Expander) Word(word string) string {
	if key, value, ok := strings.Cut(word, "="); ok && !strings.HasPrefix(word, "-") {
		return key + "=" + e.Word(value)
	}
	if strings.HasPrefix(word, "~/") || strings.HasPrefix(word, "{") {
		return e.Expand(word)
	}
	return word
}

func (e Expander) Display(abs string) string {
	if rest, ok := strings.CutPrefix(abs, e.Root+"/"); ok && e.Root != "" {
		return "/" + rest
	}
	if abs == e.Home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(abs, e.Home+"/"); ok {
		return "~/" + rest
	}
	return abs
}

func LineValue(text, key string) (string, bool) {
	for _, line := range strings.Split(text, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

func TableValue(text, key string) (string, bool) {
	cut := strings.LastIndex(key, ".")
	table, name := key[:cut], key[cut+1:]
	current := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok && current == table && strings.TrimSpace(k) == name {
			return strings.Trim(strings.TrimSpace(v), `"`), true
		}
	}
	return "", false
}
