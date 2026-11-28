package project

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const movedSentence = "spwn/ is the old project layout: its folders now sit at the project's root. Move each one up (mv spwn/agents agents, and the same for skills, tools, hooks, commands, profiles), point knowledge: at its new path, change any .gitignore line naming spwn/, then run again."

func TestALayoutRefusesEveryFolderThatIsNoPathOfTheProject(t *testing.T) {
	for _, tc := range []struct {
		folders map[string]string
		key     string
		says    string
	}{
		{map[string]string{"agents": "."}, "agents", `".", which leaves its own path by .`},
		{map[string]string{"agents": ".."}, "agents", "which leaves its own path by .."},
		{map[string]string{"skills": "a/../b"}, "skills", "which leaves its own path by .."},
		{map[string]string{"tools": ".git"}, "tools", "a hidden name, .git"},
		{map[string]string{"hooks": ".spwn"}, "hooks", "a hidden name, .spwn"},
		{map[string]string{"commands": "/etc"}, "commands", "an absolute path"},
		{map[string]string{"profiles": "~/p"}, "profiles", "which starts with ~, never expanded"},
		{map[string]string{"agents": "c:x"}, "agents", "which holds a colon"},
		{map[string]string{"agents": ""}, "agents", "empty"},
		{map[string]string{"agents": "a\nb"}, "agents", "which holds a control character"},
		{map[string]string{"agents": "Agents"}, "agents", `whose name "Agents" is not a slug`},
		{map[string]string{"agents": "spwn/agents"}, "agents", "the old layout's folder, spwn/"},
		{map[string]string{"workflows": "flows"}, "workflows", "is not a kind"},
		{map[string]string{"agents": "skills"}, "agents", "which meets skills/, the folder of skills"},
		{map[string]string{"agents": "skills/agents"}, "agents", "which meets skills/, the folder of skills"},
		{map[string]string{"skills": "agents/skills"}, "skills", "which meets agents/, the folder of agents"},
	} {
		_, err := NewLayout(tc.folders)
		var refused *LayoutError
		if !errors.As(err, &refused) || refused.Key != tc.key || !strings.Contains(refused.Message(), tc.says) {
			t.Errorf("NewLayout(%q) = %v; want folders.%s refused as %q", tc.folders, err, tc.key, tc.says)
		}
		if refused != nil && refused.Code() != LayoutCode {
			t.Errorf("NewLayout(%q) code = %q; want %q", tc.folders, refused.Code(), LayoutCode)
		}
	}
}

func TestALayoutRenamesTheOneFolderItNamesAndKeepsTheRest(t *testing.T) {
	l, err := NewLayout(map[string]string{"agents": "team/agents"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"team/agents", "skills", "tools", "hooks", "commands", "profiles"}
	if got := l.Folders(); !reflect.DeepEqual(got, want) {
		t.Errorf("Folders = %v; want %v", got, want)
	}
	if got := l.Dir("/p", KindAgents); got != filepath.Join("/p", "team", "agents") {
		t.Errorf("Dir(agents) = %q", got)
	}
	if got := DefaultLayout().Folders(); !reflect.DeepEqual(got, []string{"agents", "skills", "tools", "hooks", "commands", "profiles"}) {
		t.Errorf("DefaultLayout = %v", got)
	}
}

func projectOf(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		at := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestAProjectOfTheOldLayoutIsRefusedWithTheSentenceThatMovesIt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		said  string
	}{
		{"its folders under spwn/", map[string]string{"spwn.yaml": "version: 1\n", "spwn/agents/neo/agent.yaml": "name: neo\n"}, movedSentence},
		{"its hooks.yaml under spwn/", map[string]string{"spwn.yaml": "version: 1\n", "spwn/hooks.yaml": "{}\n"}, movedSentence},
		{"a folder at both places", map[string]string{"spwn.yaml": "version: 1\n", "spwn/skills/a.md": "a\n", "spwn/tools/t/tool.yaml": "x\n", "skills/b.md": "b\n", "tools/u/tool.yaml": "y\n"},
			movedSentence + " skills/, tools/ already sit at the root: merge what spwn/ holds of each into it, then remove spwn/."},
		{"one folder at both places", map[string]string{"spwn.yaml": "version: 1\n", "spwn/agents/neo/agent.yaml": "x\n", "agents/trinity/agent.yaml": "y\n"},
			movedSentence + " agents/ already sits at the root: merge what spwn/ holds of each into it, then remove spwn/."},
		{"a broken spwn.yaml", map[string]string{"spwn.yaml": "folders: [\n", "spwn/agents/neo/agent.yaml": "x\n"}, movedSentence},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := projectOf(t,tc.files)
			_, err := LayoutAt(root)
			var old *OldLayoutError
			if !errors.As(err, &old) || old.Message() != tc.said {
				t.Fatalf("LayoutAt = %v; want %q", err, tc.said)
			}
			if old.Code() != OldLayoutCode || !strings.HasPrefix(err.Error(), OldLayoutCode+": ") {
				t.Errorf("code = %q, error = %q; want %s", old.Code(), err.Error(), OldLayoutCode)
			}
			if _, err := Load(filepath.Join(root, "spwn.yaml")); !errors.As(err, &old) {
				t.Errorf("Load = %v; want the old layout refused", err)
			}
			if _, err := Within(root, func() (bool, error) { t.Error("act ran over the old layout"); return true, nil }); !errors.As(err, &old) {
				t.Errorf("Within = %v; want the old layout refused", err)
			}
			var paths []string
			for p := range tc.files {
				paths = append(paths, p)
			}
			if _, err := LayoutOfFiles([]byte(tc.files["spwn.yaml"]), paths); !errors.As(err, &old) || old.Message() != tc.said {
				t.Errorf("LayoutOfFiles = %v; want %q", err, tc.said)
			}
		})
	}
}

func TestASpwnAtTheRootThatIsALinkIsRefusedUnfollowed(t *testing.T) {
	for name, target := range map[string]func(t *testing.T) string{
		"to a folder of the old layout": func(t *testing.T) string { return projectOf(t,map[string]string{"agents/neo/agent.yaml": "x\n"}) },
		"to an empty folder":            func(t *testing.T) string { return t.TempDir() },
		"to nothing":                    func(t *testing.T) string { return filepath.Join(t.TempDir(), "gone") },
	} {
		t.Run(name, func(t *testing.T) {
			root := projectOf(t,map[string]string{"spwn.yaml": "version: 1\n"})
			if err := os.Symlink(target(t), filepath.Join(root, "spwn")); err != nil {
				t.Fatal(err)
			}
			_, err := LayoutAt(root)
			var old *OldLayoutError
			if !errors.As(err, &old) || old.Message() != movedSentence {
				t.Errorf("LayoutAt = %v; want the link refused with %q", err, movedSentence)
			}
		})
	}
}

func TestASpwnThatHoldsNoneOfTheOldLayoutIsNotIt(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"a file named spwn":            {"spwn.yaml": "version: 1\n", "spwn": "a binary\n"},
		"a folder of other things":     {"spwn.yaml": "version: 1\n", "spwn/notes.md": "n\n", "spwn/worlds/w/x": "x\n"},
		"no spwn at all":               {"spwn.yaml": "version: 1\n"},
		"no spwn.yaml yet, no folders": {"README.md": "r\n"},
	} {
		t.Run(name, func(t *testing.T) {
			l, err := LayoutAt(projectOf(t,files))
			if err != nil || !reflect.DeepEqual(l.Folders(), DefaultLayout().Folders()) {
				t.Errorf("LayoutAt = %v, %v; want the default layout", l.Folders(), err)
			}
		})
	}
}

func TestAProjectReadsTheFoldersItsSpwnYamlRenames(t *testing.T) {
	root := projectOf(t,map[string]string{
		"spwn.yaml":                    "version: 1\nfolders:\n  agents: crew\nworlds:\n  w:\n    agents: [neo]\n",
		"crew/neo/agent.yaml":          "name: neo\n",
		"crew/neo/AGENTS.md":           "neo\n",
		"agents/stray/agent.yaml":      "name: stray\n",
		"skills/focus.md":              "f\n",
	})
	l, err := LayoutAt(root)
	if err != nil || l.Folder(KindAgents) != "crew" {
		t.Fatalf("LayoutAt = %v, %v; want agents at crew", l.Folders(), err)
	}
	p, err := Load(filepath.Join(root, "spwn.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Agents) != 1 || p.Agents[0].Name != "neo" || !p.Agents[0].Exists || p.Agents[0].Path != filepath.Join(root, "crew", "neo") {
		t.Errorf("Agents = %+v; want neo, read from crew/neo", p.Agents)
	}
	if _, err := LayoutAt(projectOf(t,map[string]string{"spwn.yaml": "folders:\n  agents: ../away\n"})); err == nil {
		t.Error("LayoutAt of folders.agents: ../away = nil; want it refused")
	}
}
