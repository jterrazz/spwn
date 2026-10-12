package architect

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/transpile"
)

func parsePlaybookHeader(body []byte) (transpile.PlaybookEntry, bool) {
	r := bufio.NewReader(bytes.NewReader(body))
	first, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
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
