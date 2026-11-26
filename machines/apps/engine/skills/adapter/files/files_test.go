package files_test

import (
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
