package herdr_test

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/herdr"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type core struct {
	// value: a recorded herdr-core: its terminals, what each shows and
	// says, the machines it reaches, and every command line hc ran.
	terminals []map[string]any
	opened    []string
	typed     []contracttest.Typed
	machines  []string
	attached  []string
	lines     []string
}

func (c *core) answer(id, result any) local.Output {
	line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	return local.Output{Stdout: string(line) + "\n"}
}

func (c *core) connect(stdin string) local.Output {
	var req struct {
		ID     any             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(stdin)), &req) != nil {
		return local.Output{ExitCode: 2, Stderr: "not a request"}
	}
	var p struct {
		Argv     []string          `json:"argv"`
		Cwd      string            `json:"cwd"`
		Labels   map[string]string `json:"labels"`
		Terminal string            `json:"terminal"`
		Text     string            `json:"text"`
	}
	_ = json.Unmarshal(req.Params, &p)
	switch req.Method {
	case "terminal.run":
		address := "t" + strconv.Itoa(len(c.terminals)+1)
		c.terminals = append(c.terminals, map[string]any{"address": address, "cwd": p.Cwd, "state": "live", "labels": p.Labels})
		c.opened = append(c.opened, strings.TrimSpace(p.Cwd+" "+strings.Join(p.Argv, " ")))
		return c.answer(req.ID, map[string]any{"address": address})
	case "terminal.send":
		c.typed = append(c.typed, contracttest.Typed{Terminal: p.Terminal, Text: p.Text})
		return c.answer(req.ID, map[string]any{"turn": 1})
	case "terminal.keep", "terminal.kill", "terminal.rm":
		i := slices.IndexFunc(c.terminals, func(t map[string]any) bool { return t["address"] == p.Terminal })
		if i < 0 {
			line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": "not_found", "data": map[string]string{"kind": "not_found"}}})
			return local.Output{Stdout: string(line) + "\n"}
		}
		switch req.Method {
		case "terminal.kill":
			c.terminals[i]["state"] = "finished"
		case "terminal.rm":
			c.terminals = slices.Delete(c.terminals, i, i+1)
		}
		return c.answer(req.ID, map[string]any{"address": p.Terminal})
	case "terminal.list":
		other := map[string]any{"address": "t99", "cwd": "/", "state": "live", "labels": map[string]string{}}
		remote := map[string]any{"address": "mini/t1", "cwd": "/", "state": "live", "labels": map[string]string{"station": "agent"}}
		return c.answer(req.ID, map[string]any{"terminals": append(slices.Clone(c.terminals), other, remote), "hidden_finished": 0, "machines": []any{}})
	}
	line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32601, "message": "method_not_found"}})
	return local.Output{Stdout: string(line) + "\n"}
}

func (c *core) play(cmd local.Cmd) (local.Output, error) {
	c.lines = append(c.lines, strings.Join(append([]string{cmd.Name}, cmd.Args...), " "))
	args := cmd.Args
	switch {
	case len(args) == 1 && args[0] == "connect":
		return c.connect(cmd.Stdin), nil
	case len(args) >= 2 && args[0] == "machine" && args[1] == "ls":
		var list []map[string]string
		for _, m := range c.machines {
			list = append(list, map[string]string{"name": m, "state": "connected"})
		}
		data, _ := json.Marshal(map[string]any{"machines": list})
		return local.Output{Stdout: string(data)}, nil
	case len(args) >= 2 && args[0] == "machine" && args[1] == "add":
		name := args[len(args)-1]
		if slices.Contains(c.machines, name) {
			return local.Output{ExitCode: 1, Stderr: "already_added"}, nil
		}
		if !slices.Contains(args, "--install") || !slices.Contains(args, "--name") {
			return local.Output{ExitCode: 1, Stderr: "hc_missing"}, nil
		}
		c.machines = append(c.machines, name)
		return local.Output{Stdout: "{}"}, nil
	case len(args) >= 2 && args[0] == "machine" && args[1] == "rm":
		name := args[len(args)-1]
		if !slices.Contains(c.machines, name) {
			return local.Output{ExitCode: 1, Stderr: "machine_unknown"}, nil
		}
		c.machines = slices.DeleteFunc(c.machines, func(m string) bool { return m == name })
		return local.Output{Stdout: "{}"}, nil
	case len(args) == 2 && args[0] == "attach":
		c.attached = append(c.attached, args[1])
		return local.Output{}, nil
	}
	return local.Output{ExitCode: 2, Stderr: "unknown command"}, nil
}

func (c *core) says(terminal string, seen agents.Seen, message string) {
	for _, t := range c.terminals {
		if t["address"] == terminal {
			t["status"] = map[string]string{"state": string(seen), "msg": message}
		}
	}
}

func TestSession(t *testing.T) {
	contracttest.Session(t, func(*testing.T) contracttest.SessionSubject {
		c := &core{}
		return contracttest.SessionSubject{
			Session: herdr.NewSession(local.Played(c.play, "hc")),
			Opened:  func() []string { return c.opened },
			Typed:   func() []contracttest.Typed { return c.typed },
			Lines:   func() []string { return c.lines },
			Says:    c.says,
		}
	})
}

func TestMultiplexer(t *testing.T) {
	contracttest.Multiplexer(t, func(*testing.T) contracttest.MultiplexerSubject {
		c := &core{}
		return contracttest.MultiplexerSubject{
			Multiplexer: herdr.NewMultiplexer(local.Played(c.play, "hc")),
			Reached:     func() []string { return c.machines },
			Attached:    func() []string { return c.attached },
		}
	})
}

func TestAnAgentIsOpenedMarkedAsStationsAndReadyBeforeItsPrompt(t *testing.T) {
	c := &core{}
	var asked map[string]any
	run := local.Played(func(cmd local.Cmd) (local.Output, error) {
		if cmd.Name == "hc" && len(cmd.Args) == 1 {
			var req map[string]any
			_ = json.Unmarshal([]byte(cmd.Stdin), &req)
			asked, _ = req["params"].(map[string]any)
		}
		return c.play(cmd)
	}, "hc")
	if _, err := herdr.NewSession(run).Open(context.Background(), "claudecode", "linux-1"+agents.Nth+"2", "/home/admin/app", []string{"agent"}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	labels, _ := asked["labels"].(map[string]any)
	if labels["station"] != "agent" || labels["station.app"] != "claudecode" || labels["station.agent"] != "linux-1"+agents.Nth+"2" {
		t.Errorf("labels %v, want station=agent, the app and the agent's name", labels)
	}
	if asked["name"] != "linux-1-2" {
		t.Errorf("name %v, want the agent's in Herdr's letters", asked["name"])
	}
	if asked["wait"] != true || asked["no_parent"] != true || asked["outlive_parent"] != true || asked["keep"] != true {
		t.Errorf("run %v, want it waited for, parentless, outliving its caller and kept", asked)
	}
}

func TestAnAnswerOfAnErrorIsTheError(t *testing.T) {
	run := local.Played(func(local.Cmd) (local.Output, error) {
		return local.Output{Stdout: `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"blocked","data":{"kind":"blocked"}}}` + "\n"}, nil
	}, "hc")
	err := herdr.NewSession(run).Send(context.Background(), "t1", "yes")
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Errorf("Send: %v, want the core's refusal", err)
	}
}

func TestAMacWithoutHcReachesNoMachineAndForgetsNothing(t *testing.T) {
	m := herdr.NewMultiplexer(local.Played(func(local.Cmd) (local.Output, error) { return local.Output{}, nil }))
	if got, err := m.Machines(context.Background()); err != nil || got != nil {
		t.Errorf("machines %q (%v), want none and no error", got, err)
	}
	if err := m.Remove(context.Background(), "linux-1"); err != nil {
		t.Errorf("Remove: %v, want nothing to forget", err)
	}
}
