package files_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func skillAt(home string) files.Skill {
	return files.NewSkill(local.New(nil, nil, nil, nil).FS,
		filepath.Join(home, ".agents", "skills", "spwn", "SKILL.md"),
		filepath.Join(home, ".claude", "skills", "spwn", "SKILL.md"))
}

func TestTheSpwnSkillFilesKeepTheSkill(t *testing.T) {
	contracttest.Skill(t, func(t *testing.T) contract.Skill { return skillAt(t.TempDir()) })
}

func TestTheSpwnSkillLandsInEveryAgentsFolder(t *testing.T) {
	home := t.TempDir()
	if _, err := skillAt(home).Write(context.Background(), []byte("# spwn\n")); err != nil {
		t.Fatal(err)
	}
	for _, at := range []string{".agents/skills/spwn/SKILL.md", ".claude/skills/spwn/SKILL.md"} {
		if got, err := os.ReadFile(filepath.Join(home, at)); err != nil || string(got) != "# spwn\n" {
			t.Errorf("%s = %q, %v", at, got, err)
		}
	}
}

func refusedAsALink(t *testing.T, err error, at string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), at+": is a link") {
		t.Errorf("the write = %v; want %s refused as a link", err, at)
	}
}

func TestASkillFileThatIsALinkIsRefusedAndLeftAsItIs(t *testing.T) {
	home := t.TempDir()
	fsys := local.New(nil, nil, nil, nil).FS
	mine := filepath.Join(home, "dotfiles", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(mine), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mine, []byte("# mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".claude", "skills", "spwn", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsys.Symlink(mine, link); err != nil {
		t.Fatal(err)
	}
	changed, err := skillAt(home).Write(context.Background(), []byte("# spwn\n"))
	refusedAsALink(t, err, link)
	if target, err := fsys.Readlink(link); err != nil || target != mine {
		t.Errorf("the link is now %q, %v; want it kept to %s", target, err, mine)
	}
	if got, err := os.ReadFile(mine); err != nil || string(got) != "# mine\n" {
		t.Errorf("the link's own file = %q, %v; want it untouched", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(home, ".agents", "skills", "spwn", "SKILL.md")); err != nil || string(got) != "# spwn\n" || !changed {
		t.Errorf("the other copy = %q, %v, changed %v; want it written all the same", got, err, changed)
	}
}

func TestASkillFolderThatIsALinkIsNeverFollowedAndTheFailureNamesIt(t *testing.T) {
	home := t.TempDir()
	fsys := local.New(nil, nil, nil, nil).FS
	elsewhere := filepath.Join(home, "dotfiles", "spwn")
	if err := os.MkdirAll(elsewhere, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o750); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(home, ".claude", "skills", "spwn")
	if err := fsys.Symlink(elsewhere, folder); err != nil {
		t.Fatal(err)
	}
	_, err := skillAt(home).Write(context.Background(), []byte("# spwn\n"))
	refusedAsALink(t, err, folder)
	if _, err := os.ReadFile(filepath.Join(elsewhere, "SKILL.md")); err == nil {
		t.Error("the skill landed in the folder the link points at")
	}
	if got, err := os.ReadFile(filepath.Join(home, ".agents", "skills", "spwn", "SKILL.md")); err != nil || string(got) != "# spwn\n" {
		t.Errorf("the first copy = %q, %v; want it written", got, err)
	}
}

func TestAHandWrittenSkillFileIsOverwritten(t *testing.T) {
	home := t.TempDir()
	at := filepath.Join(home, ".agents", "skills", "spwn", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("# my notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := skillAt(home).Write(context.Background(), []byte("# spwn\n")); err != nil || !changed {
		t.Fatalf("the write = %v, %v", changed, err)
	}
	if got, err := os.ReadFile(at); err != nil || string(got) != "# spwn\n" {
		t.Errorf("%s = %q, %v; want the skill whole", at, got, err)
	}
}

func TestTwoFoldersThatAreOneThroughALinkAreWrittenOnce(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".agents", "skills"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}
	fsys := local.New(nil, nil, nil, nil).FS
	if err := fsys.Symlink(filepath.Join(home, ".agents", "skills"), filepath.Join(home, ".claude", "skills")); err != nil {
		t.Fatal(err)
	}
	s := skillAt(home)
	if _, err := s.Write(context.Background(), []byte("# spwn\n")); err != nil {
		t.Fatal(err)
	}
	if target, err := fsys.Readlink(filepath.Join(home, ".claude", "skills")); err != nil || target == "" {
		t.Errorf("the linked folder is no link any more: %q, %v", target, err)
	}
	if changed, err := s.Write(context.Background(), []byte("# spwn\n")); err != nil || changed {
		t.Errorf("the same skill again = %v, %v; want nothing written", changed, err)
	}
}
