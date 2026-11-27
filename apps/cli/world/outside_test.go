package world

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/packages/project"
	"spwn.sh/packages/world"
)

func projectWith(t *testing.T, manifest string) (p *project.Project, root, outside string) {
	t.Helper()
	dir := t.TempDir()
	root = filepath.Join(dir, "site")
	outside = filepath.Join(dir, "outside")
	for _, d := range []string{root, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := project.Init(root, project.InitOpts{Name: "site"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := project.Find(root)
	if err != nil || p == nil {
		t.Fatalf("Find: %v", err)
	}
	return p, root, outside
}

func TestAWorldAgentThatClimbsIsRefusedTheSameWhetherItsTargetExists(t *testing.T) {
	const manifest = "version: 1\nname: site\nworlds:\n  neo:\n    agents: [\"../../../outside/x\"]\n"
	p, _, outside := projectWith(t, manifest)

	_, gone := resolveProjectWorld(p, "neo", "")
	if err := os.MkdirAll(filepath.Join(outside, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "x", "SOUL.md"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, there := resolveProjectWorld(p, "neo", "")

	if gone == nil || there == nil {
		t.Fatalf("a climbing agent name was taken: gone=%v there=%v", gone, there)
	}
	if gone.Error() != there.Error() {
		t.Fatalf("the refusal tells what is outside:\n gone:  %v\n there: %v", gone, there)
	}
	const want = `world "neo": agent name "../../../outside/x" is not one name: a name is not empty, "." or "..", and holds no /, :, \ or NUL`
	if gone.Error() != want {
		t.Fatalf("refusal = %q, want %q", gone, want)
	}
}

func TestAWorldsHostPathsThatLeaveTheProjectAreNamedResolved(t *testing.T) {
	p, root, outside := projectWith(t, "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\".\", \"home=../outside:ro\", \"away\"]\n    knowledge: ../outside\n")
	if err := os.Symlink(outside, filepath.Join(root, "away")); err != nil {
		t.Fatal(err)
	}
	pw, err := resolveProjectWorld(p, "neo", "")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range pw.Outside {
		got = append(got, r.Path+" | "+r.Access+" | "+r.Grants)
	}
	want := []string{
		outside + ` | read-write | read-write mount at /world/knowledge in world "neo"`,
		outside + ` | read-only | read-only mount at /workspaces/home in world "neo"`,
		outside + ` | read-write | read-write mount at /workspaces/away in world "neo"`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("outside =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	wantWs := []world.Workspace{{Name: "site", Path: root}, {Name: "home", Path: outside, ReadOnly: true}, {Name: "away", Path: outside}}
	if !slices.Equal(pw.Workspaces, wantWs) || pw.Knowledge != outside {
		t.Errorf("the mounts are not the resolved paths: %+v %q", pw.Workspaces, pw.Knowledge)
	}
}

func TestTheAccessAskedIsTheAccessMounted(t *testing.T) {
	for _, entry := range []string{"a=../outside/:ro/", "c=../outside/:ro :ro"} {
		p, _, _ := projectWith(t, "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\""+entry+"\"]\n")
		var refused *cliproject.DeclaredError
		if _, err := resolveProjectWorld(p, "neo", ""); !errors.As(err, &refused) || refused.Code() != cliproject.ColonCode {
			t.Errorf("%q: %v, want %s", entry, err, cliproject.ColonCode)
		}
	}
	p, _, outside := projectWith(t, "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\"b=../outside/x:ro\", \"\", \"d=../outside/y\"]\n")
	pw, err := resolveProjectWorld(p, "neo", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []world.Workspace{
		{Name: "b", Path: filepath.Join(outside, "x"), ReadOnly: true},
		{Name: "d", Path: filepath.Join(outside, "y")},
	}
	if !slices.Equal(pw.Workspaces, want) {
		t.Fatalf("mounted %+v\nwant    %+v", pw.Workspaces, want)
	}
	for i, r := range pw.Outside {
		ws := pw.Workspaces[i]
		access := "read-write"
		if ws.ReadOnly {
			access = "read-only"
		}
		if r.Path != ws.Path || r.Access != access || !strings.Contains(r.Grants, "/workspaces/"+ws.Name+" ") {
			t.Errorf("asked %+v, mounted %+v", r, ws)
		}
	}
}

func TestAForgedWorkspaceIsRefusedBeforeAnythingIsAskedOrKept(t *testing.T) {
	for name, manifest := range map[string]string{
		"an escape in the name":  "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\"x\\e[1A\\e[2K\\r    /Users/me/code/lib\\n      read-only mount at /workspaces/lib=../..\"]\n",
		"a newline in the path":  "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\"lib=../..\\n\"]\n",
		"a tab in the knowledge": "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    knowledge: \"../\\t..\"\n",
		"an escape in an agent":  "version: 1\nname: site\nworlds:\n  neo:\n    agents: [\"neo\\e[2K\"]\n",
		"a slug-less name":       "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\"Not A Slug=../..\"]\n",
	} {
		t.Run(name, func(t *testing.T) {
			p, _, _ := projectWith(t, manifest)
			t.Setenv("SPWN_HOME", filepath.Join(p.Root, "..", "spwn-home"))
			pw, err := resolveProjectWorld(p, "neo", "")
			if err == nil {
				t.Fatalf("resolved %+v", pw)
			}
			if strings.ContainsAny(err.Error(), "\x1b\r\n\t") {
				t.Errorf("the refusal prints the project's text raw: %q", err)
			}
			if _, serr := os.Lstat(cliproject.GrantsPath()); serr == nil {
				t.Error("a grant was written")
			}
		})
	}
}

func TestATildeIsRefusedNeverExpanded(t *testing.T) {
	for _, manifest := range []string{
		"version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\"~/code\"]\n",
		"version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\"code=~/code:ro\"]\n",
		"version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    knowledge: \"~\"\n",
	} {
		p, _, _ := projectWith(t, manifest)
		_, err := resolveProjectWorld(p, "neo", "")
		var refused *cliproject.DeclaredError
		if !errors.As(err, &refused) || refused.Code() != cliproject.TildeCode {
			t.Errorf("%q: %v, want %s", manifest, err, cliproject.TildeCode)
		}
	}
}

func TestAWorldNameThatClimbsIsRefused(t *testing.T) {
	p, _, _ := projectWith(t, "version: 1\nname: site\nworlds:\n  \"..\":\n    agents: []\n")
	if _, err := resolveProjectWorld(p, "..", ""); err == nil {
		t.Fatal(`world ".." was resolved`)
	}
}

func TestEveryBindSourceOfAWorldIsResolvedAgainBeforeItsContainer(t *testing.T) {
	p, root, outside := projectWith(t, "version: 1\nname: site\nworlds:\n  neo:\n    agents: []\n    workspaces: [\"sub\"]\n    knowledge: kn\n")
	for _, d := range []string{"sub", "kn"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	pw, err := resolveProjectWorld(p, "neo", "")
	if err != nil {
		t.Fatal(err)
	}
	wantKn := []cliproject.Declared{{Text: "kn", Resolved: filepath.Join(root, "kn")}}
	wantWs := []cliproject.Declared{{Text: "sub", Resolved: filepath.Join(root, "sub")}}
	if !slices.Equal(pw.DeclaredKnowledge, wantKn) || !slices.Equal(pw.DeclaredWorkspaces, wantWs) {
		t.Fatalf("declared %+v %+v, want %+v %+v", pw.DeclaredKnowledge, pw.DeclaredWorkspaces, wantKn, wantWs)
	}
	if err := pw.recheck(true)(); err != nil {
		t.Fatalf("an unchanged world was refused: %v", err)
	}

	for _, swapped := range []string{"sub", "kn"} {
		t.Run(swapped, func(t *testing.T) {
			at := filepath.Join(root, swapped)
			if err := os.Rename(at, at+".was"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, at); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Remove(at); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(at+".was", at); err != nil {
					t.Fatal(err)
				}
			})
			var changed *cliproject.ChangedError
			if err := pw.recheck(true)(); !errors.As(err, &changed) || changed.Text != swapped || changed.Now != outside {
				t.Fatalf("recheck = %v, want a ChangedError naming %q", err, swapped)
			}
			if swapped == "sub" {
				if err := pw.recheck(false)(); err != nil {
					t.Fatalf("with -w the project's workspaces are not bound, yet were rechecked: %v", err)
				}
			}
		})
	}
}
