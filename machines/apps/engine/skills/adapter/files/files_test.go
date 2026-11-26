package files_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/skills"
	"github.com/jterrazz/studio-station/apps/engine/skills/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract/contracttest"
)

func open(t *testing.T) contract.Files {
	t.Setenv("SPWN_HOME", t.TempDir())
	return files.New(local.New(nil, nil, nil, nil).FS)
}

func TestALinkOnTheWayToTheSkillsIsNeverFollowed(t *testing.T) {
	root, away := t.TempDir(), t.TempDir()
	victim := filepath.Join(away, "kept.md")
	if err := os.WriteFile(victim, []byte("---\nname: kept\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	disk := local.New(nil, nil, nil, nil).FS
	if err := disk.MkdirAll(filepath.Join(root, "spwn"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := disk.Symlink(away, filepath.Join(root, "spwn", "skills")); err != nil {
		t.Fatal(err)
	}
	skillsOf := open(t)
	if _, err := skillsOf.Create(t.Context(), root, "focus"); !errors.Is(err, local.ErrLink) {
		t.Errorf("Create through a link = %v; want ErrLink", err)
	}
	if _, err := skillsOf.Show(t.Context(), root, "kept"); !errors.Is(err, local.ErrLink) {
		t.Errorf("Show through a link = %v; want ErrLink", err)
	}
	if err := skillsOf.Delete(t.Context(), root, "kept"); !errors.Is(err, local.ErrLink) {
		t.Errorf("Delete through a link = %v; want ErrLink", err)
	}
	if _, err := skillsOf.List(t.Context(), root); !errors.Is(err, local.ErrLink) {
		t.Errorf("List through a link = %v; want ErrLink", err)
	}
	entries, err := os.ReadDir(away)
	if err != nil || len(entries) != 1 {
		t.Errorf("the folder linked to holds %v, %v; want kept.md alone", entries, err)
	}
}

func TestFiles(t *testing.T) {
	contracttest.Files(t, func(t *testing.T) (contract.Files, string) {
		return open(t), t.TempDir()
	})
}

func TestTheUsersOwnSkillsAreThoseOfNoProject(t *testing.T) {
	contracttest.Files(t, func(t *testing.T) (contract.Files, string) {
		return open(t), ""
	})
}

func TestAProjectKeepsItsSkillsUnderItsSpwnFolder(t *testing.T) {
	root := t.TempDir()
	made, err := open(t).Create(t.Context(), root, "focus")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "spwn", "skills", "focus"+skills.Suffix); made.Path != want {
		t.Errorf("Path = %q; want %q", made.Path, want)
	}
	if _, err := os.ReadFile(made.Path); err != nil {
		t.Errorf("the skill's file is not on disk: %v", err)
	}
}

func TestOnlyMarkdownFilesAreSkills(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "spwn", "skills")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"paper-reading.md": "# Paper Reading", "notes.txt": "ignore"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	list, err := open(t).List(t.Context(), root)
	if err != nil || len(list) != 1 || list[0].Name != "paper-reading" {
		t.Errorf("List = %+v, %v; want paper-reading alone", list, err)
	}
}
