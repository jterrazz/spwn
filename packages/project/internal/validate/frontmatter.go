package validate

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the parsed YAML header of a markdown file — the
// shared convention for skill / prompt / playbook metadata. The
// header sits between two `---` delimiters at the very top of the
// file:
//
//	---
//	name: my-skill
//	description: one-line usage hint
//	---
//
//	# body goes here
//
// The Raw field carries the block verbatim (bytes between the
// delimiters, no surrounding ---); Keys is the parsed map with
// scalar values as strings — nested types collapse to their YAML
// source form so callers that only care about top-level strings
// (name, description) can read them without re-unmarshalling.
//
// Found is false when the file does not start with `---` on line 1;
// that is a legitimate state for markdown files that opt out of the
// convention, and callers decide whether the absence is an error.
type Frontmatter struct {
	Found bool
	Raw   string
	Keys  map[string]string
}

// ParseMarkdownFrontmatter reads `path` and extracts its YAML
// frontmatter block if present. Returns Found=false (no error) when
// the file does not start with `---`. Returns an error for I/O
// failures or malformed YAML inside a present block.
func ParseMarkdownFrontmatter(path string) (*Frontmatter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseFrontmatterReader(bufio.NewReader(f))
}

// ParseMarkdownFrontmatterBytes is the in-memory variant.
func ParseMarkdownFrontmatterBytes(data []byte) (*Frontmatter, error) {
	return parseFrontmatterReader(bufio.NewReader(bytes.NewReader(data)))
}

func parseFrontmatterReader(r *bufio.Reader) (*Frontmatter, error) {
	// Peek the first line. The convention requires the opening
	// `---` to be the very first line of the file; we reject
	// leading whitespace / BOM / comments so the format stays
	// strict and predictable.
	first, err := r.ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}
	if strings.TrimRight(first, "\r\n") != "---" {
		return &Frontmatter{Found: false}, nil
	}

	var body strings.Builder
	for {
		line, err := r.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "---" {
			break
		}
		body.WriteString(line)
		if err != nil {
			// Unterminated frontmatter block — treat as a parse
			// failure so the caller reports a clear error rather
			// than silently dropping the body on the floor.
			return nil, fmt.Errorf("unterminated frontmatter (missing closing ---)")
		}
	}

	raw := body.String()
	fm := &Frontmatter{Found: true, Raw: raw, Keys: map[string]string{}}
	if strings.TrimSpace(raw) == "" {
		return fm, nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &node); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	// Expect a mapping node as the top-level shape.
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return fm, nil
	}
	root := node.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter must be a YAML map, got %s", nodeKind(root.Kind))
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		k, v := root.Content[i], root.Content[i+1]
		if k.Kind != yaml.ScalarNode {
			continue
		}
		// Collapse scalars to their string value; everything else
		// keeps its YAML source so callers that need structured
		// access can re-parse the sub-chunk themselves.
		if v.Kind == yaml.ScalarNode {
			fm.Keys[k.Value] = v.Value
			continue
		}
		chunk, err := yaml.Marshal(v)
		if err != nil {
			continue
		}
		fm.Keys[k.Value] = strings.TrimRight(string(chunk), "\n")
	}
	return fm, nil
}

func nodeKind(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	}
	return "unknown"
}
