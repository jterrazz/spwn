package architect

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/dependency/tool"
	"spwn.sh/packages/platform"
)

type skillsOnly struct {
	name   string
	skills fs.FS
}

func (s skillsOnly) Name() string              { return s.name }
func (s skillsOnly) Version() string           { return "0.0.0" }
func (s skillsOnly) Dependencies() []string    { return nil }
func (s skillsOnly) Install() tool.InstallSpec { return tool.InstallSpec{} }
func (s skillsOnly) Verify() []string          { return nil }
func (s skillsOnly) Skills() fs.FS             { return s.skills }

func TestAToolSkillRefusedAsALinkIsReportedNotDropped(t *testing.T) {
	root := filepath.Join(t.TempDir(), "site")
	writeFiles(t, root, map[string]string{
		".env":                              outsideSecret + "\n",
		"spwn/tools/evil/skills/ok/SKILL.md": "# ok\n",
	})
	if err := os.Symlink(filepath.Join("..", "..", "..", "..", ".env"), filepath.Join(root, "spwn", "tools", "evil", "skills", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	skills, err := fs.Sub(platform.FSUnder(root, filepath.Join("spwn", "tools", "evil")), "skills")
	if err != nil {
		t.Fatal(err)
	}
	tools := []tool.Tool{skillsOnly{name: "local:evil", skills: skills}}

	got := refusedToolSkills(tools)
	want := []string{"tool local:evil: skill file SKILL.md is a link out of the tool's folder, refused: copy the file into the tool's folder"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("warnings = %q, want %q", got, want)
	}
	for _, s := range toolSkillEntries(tools) {
		for _, body := range s.Files {
			if string(body) == outsideSecret+"\n" {
				t.Fatal("a refused skill file reached the world")
			}
		}
	}
}
