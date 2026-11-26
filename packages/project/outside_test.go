package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const outsideSecret = "OUTSIDE-SECRET-ROOT"

// outsideProject inits a project at <dir>/site beside a tree no read may
// open: an agent folder whose files are unreadable, an unreadable note, a
// knowledge folder and a FIFO. It returns the project's root and the tree.
func outsideProject(t *testing.T) (root, outside string) {
	t.Helper()
	dir := t.TempDir()
	root = filepath.Join(dir, "site")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, InitOpts{Name: "site"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	outside = filepath.Join(dir, "outside")
	agent := filepath.Join(outside, "proj", "spwn", "agents", "secret")
	for _, d := range []string{filepath.Join(agent, "playbooks"), filepath.Join(agent, "journal"), filepath.Join(outside, "knowledge"), filepath.Join(outside, "watched")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{filepath.Join(agent, "AGENTS.md"), filepath.Join(agent, "SOUL.md"), filepath.Join(agent, "agent.yaml"), filepath.Join(outside, "abs.md")} {
		if err := os.WriteFile(f, []byte(outsideSecret+"\n"), 0o000); err != nil {
			t.Fatal(err)
		}
	}
	if err := syscall.Mkfifo(filepath.Join(outside, "notes.md"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, outside
}

func replaceIn(t *testing.T, path, old, new string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(data, []byte(old)) {
		t.Fatalf("%s holds no %q: %v", path, old, err)
	}
	if err := os.WriteFile(path, bytes.Replace(data, []byte(old), []byte(new), 1), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestEachPathOfTheDeclarationThatLeavesTheProjectIsSaidAndNeverOpened:
// every name or path the manifest, an agent.yaml or a prompt gives, as
// "..", an absolute path, a separator or a NUL, is an issue of its field,
// and nothing outside the project is read, listed or asked to exist: the
// issues are the same whether the outside path is there or not.
func TestEachPathOfTheDeclarationThatLeavesTheProjectIsSaidAndNeverOpened(t *testing.T) {
	const world = "    knowledge: ./spwn/knowledge"
	type edit struct{ file, old, new string }
	for name, tc := range map[string]struct {
		edits func(outside string) []edit
		path  string
		level Level
		says  string
	}{
		"agent-dotdot": {func(string) []edit { return []edit{{"spwn.yaml", "agents: [neo]", `agents: [neo, ".."]`}} },
			"spwn.yaml#worlds.neo.agents[1]", LevelError, "leaves the project"},
		"agent-climbing": {func(string) []edit {
			return []edit{{"spwn.yaml", "agents: [neo]", `agents: [neo, "../../../outside/proj/spwn/agents/secret"]`}}
		}, "spwn.yaml#worlds.neo.agents[1]", LevelError, "leaves the project"},
		"agent-absolute": {func(o string) []edit {
			return []edit{{"spwn.yaml", "agents: [neo]", fmt.Sprintf("agents: [neo, %q]", filepath.Join(o, "proj", "spwn", "agents", "secret"))}}
		}, "spwn.yaml#worlds.neo.agents[1]", LevelError, "leaves the project"},
		"agent-separator": {func(string) []edit { return []edit{{"spwn.yaml", "agents: [neo]", `agents: [neo, "neo/journal"]`}} },
			"spwn.yaml#worlds.neo.agents[1]", LevelError, "leaves the project"},
		"agent-nul": {func(string) []edit { return []edit{{"spwn.yaml", "agents: [neo]", `agents: [neo, "ne\x00o"]`}} },
			"spwn.yaml#worlds.neo.agents[1]", LevelError, "leaves the project"},
		"knowledge-absolute": {func(o string) []edit {
			return []edit{{"spwn.yaml", world, fmt.Sprintf("    knowledge: %q", filepath.Join(o, "knowledge"))}}
		}, "spwn.yaml#worlds.neo.knowledge", LevelError, "leaves the project"},
		"knowledge-climbing": {func(string) []edit { return []edit{{"spwn.yaml", world, "    knowledge: ../outside/knowledge"}} },
			"spwn.yaml#worlds.neo.knowledge", LevelError, "leaves the project"},
		"watch-path-absolute": {func(o string) []edit {
			return []edit{{"spwn.yaml", world, world + fmt.Sprintf("\n    automations:\n      watch:\n        on:\n          fs:\n            path: %q\n        agent: neo\n        prompt: go", filepath.Join(o, "watched"))}}
		}, "spwn.yaml#worlds.neo.automations.watch.on.fs.path", LevelInfo, "outside the project: not checked"},
		"watch-path-climbing": {func(string) []edit {
			return []edit{{"spwn.yaml", world, world + "\n    automations:\n      watch:\n        on:\n          fs:\n            path: ../outside/watched\n        agent: neo\n        prompt: go"}}
		}, "spwn.yaml#worlds.neo.automations.watch.on.fs.path", LevelInfo, "outside the project: not checked"},
		"manifest-dependency": {func(string) []edit {
			return []edit{{"spwn.yaml", "worlds:", "dependencies: [\"tool/..\"]\n\nworlds:"}}
		}, "spwn.yaml#deps", LevelError, "is invalid"},
		"agent-dependency": {func(string) []edit {
			return []edit{{"spwn/agents/neo/agent.yaml", `  - "spwn:unix"`, `  - "spwn:unix"` + "\n" + `  - "skill/.."`}}
		}, "spwn/agents/neo/agent.yaml#deps", LevelError, "is invalid"},
		"import-climbing": {func(string) []edit {
			return []edit{{"spwn/agents/neo/AGENTS.md", "\n", "\nread @../../../../outside/notes.md\n"}}
		}, "spwn/agents/neo/AGENTS.md @../../../../outside/notes.md", LevelError, "leaves the project"},
		"import-absolute": {func(o string) []edit {
			return []edit{{"spwn/agents/neo/AGENTS.md", "\n", "\nread @" + filepath.Join(o, "abs.md") + "\n"}}
		}, "", LevelError, "@-import"},
	} {
		t.Run(name, func(t *testing.T) {
			var seen [2][]Issue
			var said [2]string
			for i, there := range []bool{true, false} {
				root, outside := outsideProject(t)
				if !there {
					if err := os.RemoveAll(outside); err != nil {
						t.Fatal(err)
					}
				}
				for _, e := range tc.edits(outside) {
					replaceIn(t, filepath.Join(root, e.file), e.old, e.new)
				}
				p, err := Load(filepath.Join(root, "spwn.yaml"))
				if err != nil {
					t.Fatalf("Load: %v", err)
				}
				for _, a := range p.Agents {
					if a.Name != "neo" && (a.Exists || a.Path != "") {
						t.Errorf("agent %q resolved to %s; a name that is not one folder names none", a.Name, a.Path)
					}
				}
				seen[i] = Validate(p)
				for _, is := range seen[i] {
					if strings.Contains(is.Message, "permission denied") || strings.Contains(is.Message, outsideSecret) {
						t.Errorf("an issue read outside the project: %+v", is)
					}
				}
				said[i] = strings.ReplaceAll(fmt.Sprint(seen[i]), filepath.Dir(root), "DIR")
			}
			found := false
			for _, is := range seen[0] {
				if is.Level == tc.level && (tc.path == "" || is.Path == tc.path) && strings.Contains(is.Message, tc.says) {
					found = true
				}
			}
			if !found {
				t.Errorf("issues = %+v; want %s at %q saying %q", seen[0], tc.level, tc.path, tc.says)
			}
			if said[0] != said[1] {
				t.Errorf("the issues tell whether the outside path is there:\n there: %s\n not:   %s", said[0], said[1])
			}
		})
	}
}
