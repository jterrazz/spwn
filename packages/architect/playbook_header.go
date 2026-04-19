package architect

import (
	"bufio"
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/transpile"
)

// parsePlaybookHeader extracts the `name:` + `description:` pair
// from a playbook's YAML frontmatter. Returns ok=false when the
// block is missing, unterminated, unparseable, or lacks either
// required key.
//
// Duplicated from packages/transpile/source to avoid the
// transpile/source import cycle (transpile/source imports transpile,
// architect imports both transpile and a lot more). The parser is
// ~30 lines; the duplication is cheaper than reshuffling the layer
// graph.
func parsePlaybookHeader(body []byte) (transpile.PlaybookEntry, bool) {
	r := bufio.NewReader(bytes.NewReader(body))
	first, err := r.ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		return transpile.PlaybookEntry{}, false
	}
	if strings.TrimRight(first, "\r\n") != "---" {
		return transpile.PlaybookEntry{}, false
	}
	var block strings.Builder
	for {
		line, err := r.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "---" {
			break
		}
		block.WriteString(line)
		if err != nil {
			return transpile.PlaybookEntry{}, false
		}
	}
	var m map[string]string
	if err := yaml.Unmarshal([]byte(block.String()), &m); err != nil {
		return transpile.PlaybookEntry{}, false
	}
	name := strings.TrimSpace(m["name"])
	desc := strings.TrimSpace(m["description"])
	if name == "" || desc == "" {
		return transpile.PlaybookEntry{}, false
	}
	return transpile.PlaybookEntry{Name: name, Description: desc}, true
}
