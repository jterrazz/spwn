package source

import (
	"bufio"
	"bytes"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/transpile"
)

// promotedPlaybooks walks the agent's playbook layer and returns the
// ones that carry valid `name:` + `description:` frontmatter, ready
// to surface in CLAUDE.md as a discoverability index. Playbooks with
// no frontmatter, malformed frontmatter, or missing keys are skipped
// silently — they stay internal until the agent adds the header.
// The full authoring rule (warn on malformed blocks) lives in
// `spwn check`; this function is tolerant because a CLI dry-run
// shouldn't explode on half-written drafts.
//
// Results are sorted by Name so CLAUDE.md output is deterministic
// across map-iteration runs.
func promotedPlaybooks(playbooks map[string][]byte) []transpile.PlaybookEntry {
	out := make([]transpile.PlaybookEntry, 0, len(playbooks))
	for _, body := range playbooks {
		entry, ok := parsePlaybookHeader(body)
		if !ok {
			continue
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// parsePlaybookHeader extracts the name/description pair from a
// playbook's YAML frontmatter. Returns ok=false when the block is
// missing, unterminated, unparseable, or lacks either required key.
// This mirrors the shape the skill-frontmatter rule enforces under
// spwn/skills/ so authors carry one mental model across skills and
// promoted playbooks.
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
