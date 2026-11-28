package project

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"spwn.sh/packages/platform"
)

func skillsProject(t *testing.T, files map[string]string) (string, *platform.Tree) {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		at := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tree := platform.Within(root)
	t.Cleanup(func() { tree.Close() })
	return root, tree
}

func skillNames(skills []Skill) []string {
	names := []string{}
	for _, s := range skills {
		names = append(names, s.Name)
	}
	return names
}

func TestASkillIsOnlyWhatItsFrontMatterNames(t *testing.T) {
	root, tree := skillsProject(t, map[string]string{
		"skills/notes.md":             "# Notes\n",
		"skills/vendor/readme.txt":    "vendored\n",
		"skills/README.md":            "---\nname: readme\n---\n",
		"skills/broken.md":            "---\nname: [\n---\n",
		"skills/unnamed/SKILL.md":     "---\ndescription: no name\n---\n",
		"skills/docs/SKILL.md":        "Just prose.\n",
		"skills/review/SKILL.md":      "---\nname: review\ndescription: review\n---\nReview.\n",
		"skills/review/refs/a.md":     "a\n",
		"skills/quick.md":             "---\nname: quick\n---\nQuick.\n",
		"skills/.hidden/SKILL.md":     "---\nname: hidden\n---\n",
		"skills/vendor/lib/SKILL.md":  "---\nname: deep\n---\n",
		"skills/vendor/lib/other.txt": "x\n",
	})
	skills, err := LoadSkills(tree, filepath.Join(root, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if got := skillNames(skills); !reflect.DeepEqual(got, []string{"quick", "review"}) {
		t.Fatalf("skills = %v; want quick and review only", got)
	}
	want := map[string][]byte{
		"SKILL.md":  []byte("---\nname: review\ndescription: review\n---\nReview.\n"),
		"refs/a.md": []byte("a\n"),
	}
	if !reflect.DeepEqual(skills[1].Files, want) {
		t.Errorf("review = %q; want its files as they are", skills[1].Files)
	}
}

func TestAMissingSkillsFolderHoldsNoSkill(t *testing.T) {
	root, tree := skillsProject(t, map[string]string{"spwn.yaml": "version: 1\n"})
	if skills, err := LoadSkills(tree, filepath.Join(root, "skills")); err != nil || len(skills) != 0 {
		t.Fatalf("LoadSkills = %v, %v; want none", skills, err)
	}
}

func TestANamedSkillThatShipsWhatNoSkillMayIsRefusedNamingTheEntry(t *testing.T) {
	named := "---\nname: s\n---\n"
	cases := map[string]struct {
		make func(t *testing.T, root string)
		want string
	}{
		"a link": {
			make: func(t *testing.T, root string) {
				if err := os.Symlink(filepath.Join(root, "skills", "s", "SKILL.md"), filepath.Join(root, "skills", "s", "twin.md")); err != nil {
					t.Fatal(err)
				}
			},
			want: "project.skill-refused: skills/s/twin.md is a link, which a skill never ships",
		},
		"a FIFO": {
			make: func(t *testing.T, root string) {
				if err := syscall.Mkfifo(filepath.Join(root, "skills", "s", "pipe"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "project.skill-refused: skills/s/pipe is neither a file nor a folder",
		},
		"too many entries": {
			make: func(t *testing.T, root string) {
				for i := range skillEntries {
					if err := os.WriteFile(filepath.Join(root, "skills", "s", "f"+strings.Repeat("x", i)), nil, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			},
			want: "project.skill-refused: skills/s ships more than 64 entries",
		},
		"a file too large": {
			make: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "skills", "s", "big.md"), make([]byte, skillFileSize+1), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "project.skill-refused: skills/s/big.md holds more than 262144 bytes",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root, tree := skillsProject(t, map[string]string{"skills/s/SKILL.md": named})
			c.make(t, root)
			_, err := LoadSkills(tree, filepath.Join(root, "skills"))
			var refused *SkillError
			if !errors.As(err, &refused) || err.Error() != c.want {
				t.Fatalf("LoadSkills = %v; want %q", err, c.want)
			}
		})
	}
}

func TestALinkOrAFIFOAtTheTopOfTheSkillsFolderIsNoSkill(t *testing.T) {
	root, tree := skillsProject(t, map[string]string{"elsewhere/SKILL.md": "---\nname: away\n---\n"})
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "elsewhere"), filepath.Join(root, "skills", "away")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "skills", "pipe.md"), 0o644); err != nil {
		t.Fatal(err)
	}
	if skills, err := LoadSkills(tree, filepath.Join(root, "skills")); err != nil || len(skills) != 0 {
		t.Fatalf("LoadSkills = %v, %v; want none, in silence", skillNames(skills), err)
	}
}
