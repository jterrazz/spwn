package files_test

import (
	"context"
	"os"
	"path/filepath"
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
