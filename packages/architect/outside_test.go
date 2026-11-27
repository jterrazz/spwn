package architect

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"spwn.sh/packages/architect/internal/deploy"
	"spwn.sh/packages/platform"
)

const outsideSecret = "OUTSIDE-SECRET"

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		at := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func linkAll(t *testing.T, root string, links map[string]string) {
	t.Helper()
	for rel, target := range links {
		at := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, at); err != nil {
			t.Fatal(err)
		}
	}
}

type projectReads struct {
	skills, hooks, commands, soul, playbooks string
	resolved                                 string
	resolveErr                               string
}

func readProject(project string) projectReads {
	tree := platform.Within(project)
	defer tree.Close()
	home := filepath.Join(project, "spwn", "agents", "neo")
	body, err := NewCommandFileResolver(project).Resolve("command/evil")
	reads := projectReads{
		skills:    fmt.Sprintf("%q", collectRuntimeSkills(tree, nil)),
		hooks:     fmt.Sprintf("%q", loadRuntimeHooks(tree)),
		commands:  fmt.Sprintf("%q", loadRuntimeCommands(tree)),
		soul:      string(readAgentFile(tree, home, "SOUL.md")),
		playbooks: fmt.Sprintf("%q", loadAgentPlaybookIndex(tree, home)),
		resolved:  body,
	}
	if err != nil {
		reads.resolveErr = err.Error()
	}
	return reads
}

func projectLinkingOut(t *testing.T) (project, outside string) {
	t.Helper()
	dir := t.TempDir()
	project = filepath.Join(dir, "project")
	outside = filepath.Join(dir, "outside")
	writeFiles(t, outside, map[string]string{
		"secret.md":       outsideSecret + "\n",
		"hook.yaml":       "event: Stop\ncommand: echo " + outsideSecret + "\n",
		"skill/SKILL.md":  "---\nname: away\n---\n" + outsideSecret + "\n",
		"playbook/p.md":   "# " + outsideSecret + "\n",
		"agents/neo/x.md": outsideSecret + "\n",
	})
	writeFiles(t, project, map[string]string{
		"spwn.yaml":                         "version: 1\n",
		"spwn/skills/kept/SKILL.md":         "---\nname: kept\n---\nkept\n",
		"spwn/agents/neo/playbooks/.keep":   "",
		"spwn/commands/.keep":               "",
		"spwn/hooks/.keep":                  "",
		"spwn/skills/half/SKILL.md":         "---\nname: half\n---\nhalf\n",
		"spwn/agents/neo/playbooks/keep.md": "---\nname: keep\n---\n",
	})
	linkAll(t, project, map[string]string{
		"spwn/skills/notes.md":                filepath.Join(outside, "secret.md"),
		"spwn/skills/climb.md":                filepath.Join("..", "..", "..", "outside", "secret.md"),
		"spwn/skills/away":                    filepath.Join(outside, "skill"),
		"spwn/skills/half/refs/secret.md":     filepath.Join(outside, "secret.md"),
		"spwn/hooks/evil.yaml":                filepath.Join(outside, "hook.yaml"),
		"spwn/commands/evil.md":               filepath.Join(outside, "secret.md"),
		"spwn/agents/neo/SOUL.md":             filepath.Join(outside, "secret.md"),
		"spwn/agents/neo/playbooks/linked.md": filepath.Join(outside, "playbook", "p.md"),
	})
	return project, outside
}

func TestAProjectFileThatLinksOutIsNeverRead(t *testing.T) {
	project, outside := projectLinkingOut(t)
	there := readProject(project)
	if s := fmt.Sprintf("%+v", there); strings.Contains(s, outsideSecret) {
		t.Errorf("a project read carried the outside file: %s", s)
	}
	if there.resolveErr == "" {
		t.Errorf("Resolve(command/evil) = %q; want it refused", there.resolved)
	}
	if !strings.Contains(there.skills, "kept") || strings.Contains(there.skills, "half") {
		t.Errorf("skills = %s; want kept, and half dropped for its link out", there.skills)
	}
	if err := os.RemoveAll(outside); err != nil {
		t.Fatal(err)
	}
	if gone := readProject(project); gone != there {
		t.Errorf("the reads tell whether the outside is there:\n there: %+v\n gone:  %+v", there, gone)
	}
}

func TestAFIFOInAProjectNeverHoldsItsReads(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	writeFiles(t, project, map[string]string{
		"spwn.yaml":                       "version: 1\n",
		"spwn/skills/s/SKILL.md":          "---\nname: s\n---\n",
		"spwn/agents/neo/.keep":           "",
		"spwn/agents/neo/playbooks/.keep": "",
	})
	fifos := []string{
		"spwn/skills/pipe.md", "spwn/skills/s/extra.md", "spwn/hooks/pipe.yaml", "spwn/commands/evil.md",
		"spwn/agents/neo/SOUL.md", "spwn/agents/neo/playbooks/pipe.md",
	}
	for _, rel := range fifos {
		at := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(at, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := make(chan projectReads, 1)
	go func() { got <- readProject(project) }()
	select {
	case reads := <-got:
		if !strings.Contains(reads.resolveErr, platform.ErrNotAFile.Error()) {
			t.Errorf("Resolve of a FIFO = %q; want %v", reads.resolveErr, platform.ErrNotAFile)
		}
	case <-time.After(10 * time.Second):
		for _, rel := range fifos {
			if w, err := os.OpenFile(filepath.Join(project, rel), os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
				_ = w.Close()
			}
		}
		t.Fatal("a project read waited on a FIFO")
	}
}

func plainFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		body, err := os.ReadFile(p)
		files[filepath.ToSlash(rel)] = body
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestAnHonestProjectReadsAsAPlainReadDoes(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	writeFiles(t, project, map[string]string{
		"spwn.yaml":                         "version: 1\n",
		"shared/style.md":                   "shared style\n",
		"spwn/skills/review/SKILL.md":       "---\nname: review\ndescription: review\n---\nReview.\n",
		"spwn/skills/review/refs/a.md":      "a\n",
		"spwn/skills/review/refs/deep/b.sh": "#!/bin/sh\necho b\n",
		"spwn/skills/quick.md":              "Quick.\n",
		"spwn/hooks/lint.yaml":              "event: PostToolUse\nmatcher: Edit\ncommand: make lint\n",
		"spwn/commands/morning.md":          "Brief: {{ .Now }}.\n",
		"spwn/agents/neo/SOUL.md":           "I am neo.\n",
	})
	linkAll(t, project, map[string]string{
		"spwn/skills/review/style.md": filepath.Join("..", "..", "..", "shared", "style.md"),
	})
	tree := platform.Within(project)
	defer tree.Close()
	skills := loadUserSkills(tree)
	byName := map[string]map[string][]byte{}
	for _, s := range skills {
		byName[s.Name] = s.Files
	}
	if want := plainFiles(t, filepath.Join(project, "spwn", "skills", "review")); !reflect.DeepEqual(byName["review"], want) {
		t.Errorf("the nested skill = %q; want the plain read %q", byName["review"], want)
	}
	if want := ensureSkillFrontmatter([]byte("Quick.\n"), "quick"); !reflect.DeepEqual(byName["quick"], map[string][]byte{"SKILL.md": want}) {
		t.Errorf("the file skill = %q; want %q", byName["quick"], want)
	}
	if len(skills) != 2 {
		t.Errorf("skills = %d; want review and quick", len(skills))
	}
	if hooks := loadRuntimeHooks(tree); len(hooks) != 1 || hooks[0].Name != "lint" || hooks[0].Event != "PostToolUse" || hooks[0].Matcher != "Edit" || hooks[0].Command != "make lint" {
		t.Errorf("hooks = %+v; want lint", hooks)
	}
	if commands := loadRuntimeCommands(tree); len(commands) != 1 || commands[0].Name != "morning" || string(commands[0].Body) != "Brief: {{ .Now }}.\n" {
		t.Errorf("commands = %q; want morning", commands)
	}
	if body, err := NewCommandFileResolver(project).Resolve("command/morning"); err != nil || body != "Brief: {{ .Now }}.\n" {
		t.Errorf("Resolve(command/morning) = %q, %v; want its body", body, err)
	}
	if soul := readAgentFile(tree, filepath.Join(project, "spwn", "agents", "neo"), "SOUL.md"); string(soul) != "I am neo.\n" {
		t.Errorf("SOUL.md = %q; want the agent's", soul)
	}
}

func TestAnAgentsFolderThatLinksOutIsNotCopiedIn(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	outside := filepath.Join(dir, "outside")
	writeFiles(t, outside, map[string]string{"neo/SOUL.md": outsideSecret + "\n"})
	writeFiles(t, project, map[string]string{"spwn.yaml": "version: 1\n"})
	linkAll(t, project, map[string]string{"spwn/agents": outside})
	t.Setenv("SPWN_HOME", t.TempDir())
	mb := newMockBackend()
	_, err := platform.WithProjectRoot(project, func() (struct{}, error) {
		return struct{}{}, deploy.SyncIn(context.Background(), mb, "ctr", map[string]string{"neo": "/agents/neo"})
	})
	if err != nil {
		t.Fatalf("SyncIn: %v", err)
	}
	if len(mb.copyDirToCalls) != 0 {
		t.Errorf("SyncIn copied %+v from a linked agents folder; want nothing", mb.copyDirToCalls)
	}
}
