package herdr

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const TerminalVar = "HC_TERMINAL_ID"

const (
	hc = "hc"
	// value: the labels a terminal station opens carries: what it is, and
	// the app whose agent runs in it.
	markKey, markValue = "station", "agent"
	appKey             = "station.app"
	nameKey            = "station.agent"
	// value: how long an agent opened may take to be ready for its first
	// prompt; past it the prompt is typed anyway.
	ready  = 60
	budget = 10 * time.Second
	// value: an add copies hc to the machine when it has none.
	addBudget = 3 * time.Minute
)

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
	Attach(ctx context.Context, c local.Cmd, out local.Stream) (local.Attached, error)
}

type Session struct{ run Runner }

var _ contract.Session = (*Session)(nil)

func NewSession(run Runner) *Session { return &Session{run: run} }

type request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type reply struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
		Data    struct {
			Kind string `json:"kind"`
		} `json:"data"`
	} `json:"error"`
}

// invariant: every call is one request of herdr-core's API on hc's stdin,
// so a text typed to an agent is never on a command line.
func (s *Session) call(ctx context.Context, wait time.Duration, method string, params, result any) error {
	line, err := json.Marshal(request{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return err
	}
	out, err := s.run.Output(ctx, local.Cmd{Name: hc, Args: []string{"connect"}, Stdin: string(line) + "\n", Timeout: wait})
	if err != nil {
		return err
	}
	for _, l := range strings.Split(out.Stdout, "\n") {
		var r reply
		if json.Unmarshal([]byte(l), &r) != nil || r.ID != 1 {
			continue
		}
		if r.Error != nil {
			return errors.New("hc " + method + ": " + strings.TrimSpace(r.Error.Message+" "+r.Error.Data.Kind))
		}
		if result == nil {
			return nil
		}
		return json.Unmarshal(r.Result, result)
	}
	return errors.New("hc " + method + " answered nothing: " + strings.TrimSpace(out.Stderr))
}

type runParams struct {
	Argv []string `json:"argv"`
	// value: the terminal's own name in Herdr, the agent's in Herdr's letters.
	Name          string            `json:"name,omitempty"`
	Cwd           string            `json:"cwd,omitempty"`
	Labels        map[string]string `json:"labels"`
	NoParent      bool              `json:"no_parent"`
	OutliveParent bool              `json:"outlive_parent"`
	Keep          bool              `json:"keep"`
	Wait          bool              `json:"wait"`
	Timeout       float64           `json:"timeout"`
}

func (s *Session) Open(ctx context.Context, app agents.App, name, folder string, argv []string) (string, error) {
	var ran struct {
		Address string `json:"address"`
	}
	labels := map[string]string{markKey: markValue, appKey: string(app)}
	if name != "" {
		labels[nameKey] = name
	}
	err := s.call(ctx, budget+ready*time.Second, "terminal.run", runParams{
		Argv: argv, Name: agents.Key(name), Cwd: folder, Labels: labels,
		NoParent: true, OutliveParent: true, Keep: true, Wait: true, Timeout: ready,
	}, &ran)
	if err == nil && ran.Address == "" {
		err = errors.New("hc terminal.run answered no terminal")
	}
	return ran.Address, err
}

func (s *Session) Send(ctx context.Context, terminal, text string) error {
	return s.call(ctx, budget, "terminal.send", map[string]string{"terminal": terminal, "text": text}, nil)
}

func (s *Session) Close(ctx context.Context, terminal string) error {
	// invariant: a terminal is kept by Herdr, so its program is ended once
	// it is no longer kept, then the finished terminal removed; one Herdr no
	// longer holds is already closed.
	at := map[string]any{"terminal": terminal}
	for _, method := range []string{"terminal.keep", "terminal.kill", "terminal.rm"} {
		params := at
		if method == "terminal.keep" {
			params = map[string]any{"terminal": terminal, "keep": false}
		}
		if err := s.call(ctx, budget, method, params, nil); err != nil {
			if strings.Contains(err.Error(), "not_found") {
				return nil
			}
			if method != "terminal.rm" {
				return err
			}
		}
	}
	return nil
}

type terminal struct {
	Address string            `json:"address"`
	Cwd     string            `json:"cwd"`
	State   string            `json:"state"`
	Labels  map[string]string `json:"labels"`
	Status  *struct {
		State string `json:"state"`
		Msg   string `json:"msg"`
	} `json:"status"`
}

func (s *Session) Terminals(ctx context.Context) ([]agents.Terminal, error) {
	var list struct {
		Terminals []terminal `json:"terminals"`
	}
	if err := s.call(ctx, budget, "terminal.list", map[string]any{"labels": map[string]string{markKey: markValue}}, &list); err != nil {
		return nil, err
	}
	var out []agents.Terminal
	for _, t := range list.Terminals {
		if t.Labels[markKey] != markValue || strings.Contains(t.Address, "/") || t.State != "live" {
			continue
		}
		at := agents.Terminal{Address: t.Address, Name: t.Labels[nameKey], App: agents.App(t.Labels[appKey]), Folder: t.Cwd}
		if t.Status != nil {
			at.Seen, at.Message = agents.Seen(t.Status.State), t.Status.Msg
		}
		out = append(out, at)
	}
	return out, nil
}

type Multiplexer struct{ run Runner }

var _ contract.Multiplexer = (*Multiplexer)(nil)

func NewMultiplexer(run Runner) *Multiplexer { return &Multiplexer{run: run} }

func (h *Multiplexer) hc(ctx context.Context, wait time.Duration, args ...string) (local.Output, error) {
	out, err := h.run.Output(ctx, local.Cmd{Name: hc, Args: args, Timeout: wait})
	if err != nil {
		return out, err
	}
	if out.ExitCode != 0 {
		return out, errors.New("hc " + strings.Join(args[:2], " ") + ": " + strings.TrimSpace(out.Stderr+" "+out.Stdout))
	}
	return out, nil
}

func (h *Multiplexer) Machines(ctx context.Context) ([]string, error) {
	// invariant: a Mac without hc reaches no machine.
	out, err := h.hc(ctx, budget, "machine", "ls", "--json")
	if errors.Is(err, local.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list struct {
		Machines []struct {
			Name string `json:"name"`
		} `json:"machines"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &list); err != nil {
		return nil, errors.New("hc machine ls: " + err.Error())
	}
	var names []string
	for _, m := range list.Machines {
		names = append(names, m.Name)
	}
	return names, nil
}

func (h *Multiplexer) Add(ctx context.Context, machine string) error {
	names, err := h.Machines(ctx)
	if err != nil || slices.Contains(names, machine) {
		return err
	}
	_, err = h.hc(ctx, addBudget, "machine", "add", "--install", "--json", "--name", machine, machine)
	return err
}

func (h *Multiplexer) Remove(ctx context.Context, machine string) error {
	names, err := h.Machines(ctx)
	if err != nil || !slices.Contains(names, machine) {
		return err
	}
	_, err = h.hc(ctx, budget, "machine", "rm", "--json", machine)
	return err
}

func (h *Multiplexer) Attach(ctx context.Context, machine, terminal string) error {
	address := terminal
	if machine != "" {
		address = machine + "/" + terminal
	}
	ran, err := h.run.Attach(ctx, local.Cmd{Name: hc, Args: []string{"attach", address}}, local.ToStderr)
	if err == nil && ran.ExitCode != 0 {
		err = &local.ExitError{Step: "hc attach", Code: ran.ExitCode, LastStderr: ran.LastStderr}
	}
	return err
}
