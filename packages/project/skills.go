package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/platform"
)

// Skill is one skill of a project's skills folder: Name is its folder's
// name, or its file's without .md, and Files is every file it ships by
// slash path from the skill's own root, SKILL.md always among them.
type Skill struct {
	Name  string
	Files map[string][]byte
}

// SkillCode is the code of a SkillError.
const SkillCode = "project.skill-refused"

// SkillError refuses a skill whose folder ships what no skill may: a
// link, an entry that is neither a file nor a folder, or more files or
// bytes than a skill carries. Path is the entry, from the project's root.
type SkillError struct {
	Path   string
	Reason string
}

func (e *SkillError) Error() string { return SkillCode + ": " + e.Path + " " + e.Reason }

// Code is SkillCode.
func (e *SkillError) Code() string { return SkillCode }

// value: the most entries one skill folder ships, its folders counted,
// and the most bytes one of its files holds.
const (
	skillEntries  = 64
	skillFileSize = 256 << 10
)

// SkillEntry is the file of a skill, inside its folder.
const SkillEntry = "SKILL.md"

// invariant: a skill is named, never inferred from where a file sits. A
// folder under the skills folder is a skill only when it holds a
// SKILL.md file whose front matter parses and names it; a file directly
// under it is one only when it ends in .md, is not a README.md, and its
// own front matter names it. Anything else (a link, a folder of other
// files, a markdown note with no front matter) is not a skill, and is
// passed over in silence, so a repository's own skills/ folder becomes
// no prompt of any agent. A skill once named ships every file of its
// folder, and is refused, never partly shipped, when one is a link or
// past the bounds.

// LoadSkills is every skill of the skills folder dir, read through t and
// sorted by name; a folder that does not exist holds none.
func LoadSkills(t *platform.Tree, dir string) ([]Skill, error) {
	entries, err := t.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []Skill
	for _, e := range entries {
		name := e.Name()
		path := filepath.Join(dir, name)
		switch {
		case strings.HasPrefix(name, "."):
		case e.IsDir():
			body, ok := namedEntry(t, filepath.Join(path, SkillEntry))
			if !ok {
				continue
			}
			files, err := skillFolder(t, path)
			if err != nil {
				return nil, err
			}
			files[SkillEntry] = body
			out = append(out, Skill{Name: name, Files: files})
		case e.Type().IsRegular() && strings.HasSuffix(name, ".md"):
			body, _ := namedEntry(t, path)
			if !SkillFile(name, body) {
				continue
			}
			out = append(out, Skill{Name: strings.TrimSuffix(name, ".md"), Files: map[string][]byte{SkillEntry: body}})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func namedEntry(t *platform.Tree, path string) ([]byte, bool) {
	info, err := t.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > skillFileSize {
		return nil, false
	}
	body, err := t.ReadFile(path)
	if err != nil || len(body) > skillFileSize {
		return nil, false
	}
	return body, frontMatterName(body) != ""
}

// value: a file named name directly under a skills folder, holding body,
// is a skill: a .md file, neither hidden nor a README.md, no larger than a
// skill's file, whose own front matter parses and carries a name. The one
// rule of a bare skill file, for every binary that lists or loads one.

func SkillFile(name string, body []byte) bool {
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".md") && !strings.EqualFold(name, "README.md") &&
		len(body) <= skillFileSize && frontMatterName(body) != ""
}

func frontMatterName(body []byte) string {
	rest, ok := bytes.CutPrefix(bytes.TrimLeft(body, "\n\t "), []byte("---\n"))
	if !ok {
		return ""
	}
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return ""
	}
	var front struct {
		Name string `yaml:"name"`
	}
	if yaml.Unmarshal(rest[:end], &front) != nil {
		return ""
	}
	return strings.TrimSpace(front.Name)
}

func skillFolder(t *platform.Tree, root string) (map[string][]byte, error) {
	files := map[string][]byte{}
	seen := 0
	refuse := func(p, reason string) error {
		rel, err := t.Rel(p)
		if err != nil {
			rel = p
		}
		return &SkillError{Path: filepath.ToSlash(rel), Reason: reason}
	}
	err := t.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		seen++
		if seen > skillEntries {
			return refuse(root, fmt.Sprintf("ships more than %d entries", skillEntries))
		}
		switch {
		case d.Type()&os.ModeSymlink != 0:
			return refuse(p, "is a link, which a skill never ships")
		case d.IsDir():
			return nil
		case !d.Type().IsRegular():
			return refuse(p, "is neither a file nor a folder")
		}
		body, err := t.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		if len(body) > skillFileSize {
			return refuse(p, fmt.Sprintf("holds more than %d bytes", skillFileSize))
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = body
		return nil
	})
	return files, err
}
