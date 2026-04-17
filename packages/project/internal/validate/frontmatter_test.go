package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseMarkdownFrontmatter_happyPath: canonical "---\n...\n---"
// header → Found=true, Keys populated with scalar values.
func TestParseMarkdownFrontmatter_happyPath(t *testing.T) {
	body := `---
name: my-skill
description: Use when summarising papers.
---

# My Skill

Body text.
`
	fm, err := ParseMarkdownFrontmatterBytes([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !fm.Found {
		t.Fatal("expected Found=true for a well-formed header")
	}
	if got := fm.Keys["name"]; got != "my-skill" {
		t.Errorf("name: got %q, want my-skill", got)
	}
	if got := fm.Keys["description"]; got != "Use when summarising papers." {
		t.Errorf("description: got %q", got)
	}
}

// TestParseMarkdownFrontmatter_noHeader: a plain markdown file
// without a `---` header returns Found=false, no error.
func TestParseMarkdownFrontmatter_noHeader(t *testing.T) {
	fm, err := ParseMarkdownFrontmatterBytes([]byte("# Just a markdown\n\nbody\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fm.Found {
		t.Error("expected Found=false for plain markdown")
	}
}

// TestParseMarkdownFrontmatter_unterminated: an opening --- with no
// closing --- is a hard error; silent acceptance would swallow the
// body into the frontmatter and confuse downstream consumers.
func TestParseMarkdownFrontmatter_unterminated(t *testing.T) {
	_, err := ParseMarkdownFrontmatterBytes([]byte("---\nname: foo\nbody here no closer\n"))
	if err == nil {
		t.Fatal("expected error for unterminated frontmatter")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("want 'unterminated' in error, got: %v", err)
	}
}

// TestParseMarkdownFrontmatter_extraKeysKept: other keys beyond
// name/description are retained so authors can attach metadata
// (version, tags, …) without the parser choking or silently
// discarding them.
func TestParseMarkdownFrontmatter_extraKeysKept(t *testing.T) {
	body := `---
name: n
description: d
version: "6.0"
tags: [a, b]
---

body
`
	fm, err := ParseMarkdownFrontmatterBytes([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := fm.Keys["version"]; got != "6.0" {
		t.Errorf("version scalar: got %q, want 6.0", got)
	}
	if _, ok := fm.Keys["tags"]; !ok {
		t.Error("non-scalar key 'tags' should still be present in Keys")
	}
}

// TestParseMarkdownFrontmatter_emptyHeader: an empty `---\n---`
// block is legal — Found=true, Keys empty. The caller rule decides
// whether that's acceptable.
func TestParseMarkdownFrontmatter_emptyHeader(t *testing.T) {
	fm, err := ParseMarkdownFrontmatterBytes([]byte("---\n---\n\nbody\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !fm.Found {
		t.Error("empty frontmatter is still frontmatter")
	}
	if len(fm.Keys) != 0 {
		t.Errorf("want empty Keys, got %v", fm.Keys)
	}
}

// TestParseMarkdownFrontmatter_fromFile sanity-checks the file-path
// variant reads bytes identical to the in-memory one.
func TestParseMarkdownFrontmatter_fromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.md")
	if err := os.WriteFile(path, []byte("---\nname: f\ndescription: g\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ParseMarkdownFrontmatter(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !fm.Found || fm.Keys["name"] != "f" || fm.Keys["description"] != "g" {
		t.Errorf("from-file parse mismatch: %+v", fm)
	}
}

// TestParseMarkdownFrontmatter_rejectsNonMapTop: a frontmatter that
// isn't a YAML map (e.g. a bare list or scalar) is a parse error
// because downstream rules assume key-value lookup.
func TestParseMarkdownFrontmatter_rejectsNonMapTop(t *testing.T) {
	_, err := ParseMarkdownFrontmatterBytes([]byte("---\n- just\n- a\n- list\n---\n"))
	if err == nil {
		t.Fatal("expected error for non-map frontmatter")
	}
}
