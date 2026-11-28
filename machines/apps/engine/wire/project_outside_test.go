package wire_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/calls"
)

// value: a tree beside the project at folder that no call may open: an
// agent's folder whose files are unreadable, an unreadable note, a
// knowledge folder, and a FIFO, which a read would wait on. A call that
// opened any of them would say so in an error, or wait.
type canary struct{ agent, note, knowledge, fifo string }

func besideTheProject(t *testing.T, folder string) canary {
	t.Helper()
	out := filepath.Join(folder, "outside")
	c := canary{
		agent:     filepath.Join(out, "proj", "agents", "secret"),
		note:      filepath.Join(out, "abs.md"),
		knowledge: filepath.Join(out, "knowledge"),
		fifo:      filepath.Join(out, "notes.md"),
	}
	for _, dir := range []string{filepath.Join(c.agent, "playbooks"), filepath.Join(c.agent, "journal"), c.knowledge} {
		if err := machine.FS.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{filepath.Join(c.agent, "AGENTS.md"), filepath.Join(c.agent, "SOUL.md"), filepath.Join(c.agent, "agent.yaml"), c.note} {
		if err := os.WriteFile(f, []byte(secret+"\n"), 0o000); err != nil {
			t.Fatal(err)
		}
	}
	mkfifo(t, c.fifo)
	return c
}

// value: the project's spwn.yaml with old replaced by with, once.
func declaring(t *testing.T, root, old, with string) {
	t.Helper()
	at := filepath.Join(root, project.ManifestName)
	data, err := os.ReadFile(at)
	if err != nil || !bytes.Contains(data, []byte(old)) {
		t.Fatalf("spwn.yaml holds no %q: %v", old, err)
	}
	if err := os.WriteFile(at, bytes.Replace(data, []byte(old), []byte(with), 1), 0o600); err != nil {
		t.Fatal(err)
	}
}

// value: each read of the project at root, bounded by the canary's FIFO,
// and the plan the plan call returned.
func readsBounded(t *testing.T, d calls.Deps, root string, c canary) (plan project.Plan) {
	t.Helper()
	for name, read := range reads(t, d, root) {
		err := within(t, name, c.fifo, 10*time.Second, read)
		switch name {
		case "plan", "show", "drift":
			if err != nil {
				t.Errorf("%s = %v; want it read, the outside left alone", name, err)
			}
		case "render":
			if err != nil && strings.Contains(err.Error(), "permission denied") {
				t.Errorf("render = %v; it opened a file outside the project", err)
			}
		case "apply":
			if !refusedWith(err, calls.Invalid) {
				t.Errorf("apply = %v; want it refused as the plan's error", err)
			}
		}
	}
	plan, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil)
	if err != nil {
		t.Fatalf("Plan = %v", err)
	}
	shown, err := calls.Show.Run(t.Context(), d, calls.ShowIn{Folder: root}, nil)
	if err != nil || !slices.Equal(shown.Agents, []string{project.Starter}) {
		t.Errorf("Show = %v, %v; want the starter alone listed", shown.Agents, err)
	}
	for _, i := range plan.Issues {
		if strings.Contains(i.Message, "permission denied") || strings.Contains(i.Message, secret) {
			t.Errorf("the plan read outside the project: %+v", i)
		}
	}
	if _, err := machine.FS.Stat(filepath.Join(root, project.StateFolder)); err == nil {
		t.Errorf("%s is there; nothing is written while the plan has an error", project.StateFolder)
	}
	return plan
}

func leaves(plan project.Plan, path, says string) bool {
	return !plan.Valid && slices.ContainsFunc(plan.Issues, func(i project.Issue) bool {
		return i.Level == project.Error && i.Path == path && strings.Contains(i.Message, says) && strings.Contains(i.Message, "leaves the project")
	})
}

func TestADeclaredAgentOrKnowledgeThatLeavesTheProjectIsAPlanErrorAndNothingOutsideIsOpened(t *testing.T) {
	for name, field := range map[string]func(c canary) (old, with, path string){
		"agent-climbing": func(canary) (string, string, string) {
			return "agents: [neo]", `agents: [neo, "../../../outside/proj/agents/secret"]`, "spwn.yaml#worlds.neo.agents[1]"
		},
		"agent-dotdot": func(canary) (string, string, string) {
			return "agents: [neo]", `agents: [neo, ".."]`, "spwn.yaml#worlds.neo.agents[1]"
		},
		"agent-at-a-fifo": func(canary) (string, string, string) {
			return "agents: [neo]", `agents: [neo, "../../../outside/notes.md"]`, "spwn.yaml#worlds.neo.agents[1]"
		},
		"agent-absolute": func(c canary) (string, string, string) {
			return "agents: [neo]", fmt.Sprintf("agents: [neo, %q]", c.agent), "spwn.yaml#worlds.neo.agents[1]"
		},
		"agent-separator": func(canary) (string, string, string) {
			return "agents: [neo]", `agents: [neo, "neo/playbooks"]`, "spwn.yaml#worlds.neo.agents[1]"
		},
		"agent-nul": func(canary) (string, string, string) {
			return "agents: [neo]", `agents: [neo, "ne\x00o"]`, "spwn.yaml#worlds.neo.agents[1]"
		},
		"knowledge-absolute": func(c canary) (string, string, string) {
			return "knowledge: ./knowledge", fmt.Sprintf("knowledge: %q", c.knowledge), "spwn.yaml#worlds.neo.knowledge"
		},
		"knowledge-absolute-missing": func(c canary) (string, string, string) {
			return "knowledge: ./knowledge", fmt.Sprintf("knowledge: %q", c.knowledge+"-not-there"), "spwn.yaml#worlds.neo.knowledge"
		},
		"knowledge-climbing": func(canary) (string, string, string) {
			return "knowledge: ./knowledge", "knowledge: ../outside/knowledge", "spwn.yaml#worlds.neo.knowledge"
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, folder, root := applying(t)
			c := besideTheProject(t, folder)
			old, with, path := field(c)
			declaring(t, root, old, with)
			if plan := readsBounded(t, d, root, c); !leaves(plan, path, "") {
				t.Errorf("Plan = %+v; want %s an error that leaves the project", plan.Issues, path)
			}
		})
	}
}

func TestAnImportThatLeavesTheProjectIsAPlanErrorAndNeverRead(t *testing.T) {
	d, folder, root := applying(t)
	c := besideTheProject(t, folder)
	prompt := filepath.Join(root, "agents", project.Starter, "AGENTS.md")
	data, err := os.ReadFile(prompt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prompt, append(data, "\nread @../../../../outside/notes.md and @"+c.note+"\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := readsBounded(t, d, root, c)
	from := "agents/" + project.Starter + "/AGENTS.md"
	for _, target := range []string{"../../../../outside/notes.md", c.note} {
		if !leaves(plan, from+" @"+target, "@-import "+target) {
			t.Errorf("Plan = %+v; want the import of %s an error that leaves the project", plan.Issues, target)
		}
	}
}
