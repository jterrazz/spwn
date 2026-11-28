package architect

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/dependency/tool"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/world/models"
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
		".env":                          outsideSecret + "\n",
		"tools/evil/skills/ok/SKILL.md": "# ok\n",
	})
	if err := os.Symlink(filepath.Join("..", "..", "..", "..", ".env"), filepath.Join(root, "tools", "evil", "skills", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	skills, err := fs.Sub(platform.FSUnder(root, filepath.Join("tools", "evil")), "skills")
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

func TestSpawnSaysARefusedToolSkillInItsWarnings(t *testing.T) {
	isolateSpawn(t)
	root := filepath.Join(t.TempDir(), "site")
	writeFiles(t, root, map[string]string{
		".env":                          outsideSecret + "\n",
		"tools/evil/tool.yaml":          "name: evil\n",
		"tools/evil/skills/ok/SKILL.md": "# ok\n",
	})
	linkAll(t, root, map[string]string{
		"tools/evil/skills/SKILL.md": filepath.Join("..", "..", "..", "..", ".env"),
	})
	platform.SetProject(root, platform.Folders{Agents: "agents", Skills: "skills", Tools: "tools", Hooks: "hooks", Commands: "commands", Profiles: "profiles"})
	t.Cleanup(func() { platform.SetProject("", platform.Folders{}) })

	res, err := New(newMockBackend(), nil).Spawn(context.Background(), SpawnOpts{
		ConfigName: "neo",
		Manifest:   models.Manifest{Deps: []string{"tool/evil"}},
	})
	if err != nil {
		t.Fatalf("Spawn = %v", err)
	}
	for _, w := range res.Warnings {
		if strings.Contains(w, "tool local:evil: skill file SKILL.md is a link out of the tool's folder, refused") {
			return
		}
	}
	t.Fatalf("Spawn's warnings = %q, want the refused skill said", res.Warnings)
}
