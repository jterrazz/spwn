package guest

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	agentscontract "github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	KeysBegin      = "# >>> station hosts >>>"
	KeysEnd        = "# <<< station hosts <<<"
	scanEvery      = time.Second
	snapshotEach   = 60
	containersEach = 5
	relayBudget    = 30 * time.Second
	readBudget     = 10 * time.Second
)

type Options struct {
	// value: the guest's own state folder and the services' env file.
	State, PortsEnv string
	// value: the range of this system's outgoing connections' ports; nil
	// takes the default of the kind.
	Ephemeral Ephemeral
	Kind      catalog.MachineKind
	Version   string
	// value: what the host's `accounts` request is answered with: this VM's
	// providers, parsed into identities and scopes, never a token.
	Accounts func(ctx context.Context, keys []string) any
	// value: what listens in this VM, read by the adapter of its kind.
	Listeners Listeners
	// value: the ports this VM's containers publish; nil reads none.
	Containers Containers
	// value: the journal this VM's agent hooks write, the one store of the
	// agents the guest reads.
	Journal agentscontract.Journal
}

type Listeners interface {
	Read(ctx context.Context) ([]ports.Listener, error)
}

type Containers interface {
	Read(ctx context.Context) ([]ports.Announced, error)
}

type Agent struct {
	h local.System
	o Options
}

func New(h local.System, o Options) *Agent { return &Agent{h: h, o: o} }

type Hello struct {
	// value: alias of hello v1, the line the engine of any version writes.
	Mac        string            `json:"host"`
	VM         string            `json:"vm"`
	PublicName string            `json:"public_host"`
	Keys       []string          `json:"keys"`
	Env        map[string]string `json:"env"`
	Ignored    []int             `json:"ignored"`
	MinPort    int               `json:"min_port"`
}

type message struct {
	V      int             `json:"v"`
	Type   string          `json:"type"`
	ID     string          `json:"id,omitempty"`
	Op     string          `json:"op,omitempty"`
	OK     bool            `json:"ok,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
	Hello
}

func (s *Agent) SocketPath() string { return path.Join(s.o.State, "guest.sock") }

type session struct {
	s       *Agent
	out     io.Writer
	mu      sync.Mutex
	hello   Hello
	ready   bool
	from    int
	to      int
	last    []ports.Announced
	held    []ports.Announced
	pending map[string]chan message
	seq     int
}

func (g *session) send(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	_, _ = g.out.Write(append(data, '\n'))
}

func (s *Agent) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	// invariant: the guest never opens a connection to its host: its stdin and
	// stdout ARE the host's channel, and a stdin that ends ends the guest.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g := &session{s: s, out: out, pending: map[string]chan message{}}
	lines := make(chan message)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(in)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			var m message
			if json.Unmarshal(scanner.Bytes(), &m) == nil {
				select {
				case lines <- m:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	sock := s.listen(ctx, g, cancel)
	defer func() {
		if sock != nil {
			_ = sock.Close()
		}
		_ = s.h.FS.Remove(s.SocketPath())
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case m, ok := <-lines:
			if !ok {
				return nil
			}
			g.handle(ctx, m)
		}
	}
}

func (g *session) handle(ctx context.Context, m message) {
	switch m.Type {
	case "hello":
		g.mu.Lock()
		g.hello = m.Hello
		g.mu.Unlock()
		_ = g.s.writeFiles(m.Hello)
		if !g.ready {
			g.ready = true
			g.from, g.to = g.s.ephemeral(ctx)
			g.send(map[string]any{"v": 1, "type": "ready", "guest_version": g.s.o.Version, "kind": string(g.s.o.Kind)})
			g.last = g.s.scan(ctx, g.hello, &g.held, true, g.from, g.to)
			g.send(map[string]any{"v": 1, "type": "snapshot", "ports": g.last})
			go g.watch(ctx)
			g.report(ctx)
		}
	case "ping":
		g.send(map[string]any{"v": 1, "type": "pong"})
	case "request":
		go g.answer(ctx, m)
	case "reply":
		g.mu.Lock()
		ch := g.pending[m.ID]
		delete(g.pending, m.ID)
		g.mu.Unlock()
		if ch != nil {
			ch <- m
		}
	}
}

// invariant: a host's request is answered on the same channel, with only the
// parsed result: the providers' own output never leaves this process.
func (g *session) answer(ctx context.Context, m message) {
	if m.Op != "accounts" || g.s.o.Accounts == nil {
		g.send(map[string]any{"v": 1, "type": "reply", "id": m.ID, "ok": false, "error": map[string]string{"code": "unknown_op", "message": "unknown op \"" + m.Op + "\"", "hint": "station accounts"}})
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readBudget)
	defer cancel()
	g.mu.Lock()
	keys := g.hello.Keys
	g.mu.Unlock()
	g.send(map[string]any{"v": 1, "type": "reply", "id": m.ID, "ok": true, "result": g.s.o.Accounts(ctx, keys)})
}

// invariant: SPEC B5.1: the guest reports its VM's journal with its first
// snapshot, then again on each change, whole; it never reads the
// journal's folder itself.
func (g *session) report(ctx context.Context) {
	if g.s.o.Journal == nil {
		return
	}
	changes, err := g.s.o.Journal.Watch(ctx)
	if err != nil {
		return
	}
	send := func(kept any) { g.send(map[string]any{"v": 1, "type": "agents", "agents": kept}) }
	if first, ok := <-changes; ok {
		send(first)
	}
	go func() {
		for kept := range changes {
			send(kept)
		}
	}()
}

func (g *session) watch(ctx context.Context) {
	for tick := 1; ; tick++ {
		if g.s.h.Clock.Sleep(ctx, scanEvery) != nil {
			return
		}
		now := g.s.scan(ctx, g.hello, &g.held, tick%containersEach == 0, g.from, g.to)
		if tick%snapshotEach == 0 {
			g.send(map[string]any{"v": 1, "type": "snapshot", "ports": now})
		} else {
			for _, e := range ports.DiffAnnounced(g.last, now) {
				if e.Listen != nil {
					g.send(struct {
						V    int    `json:"v"`
						Type string `json:"type"`
						ports.Announced
					}{1, "listen", *e.Listen})
				} else {
					g.send(map[string]any{"v": 1, "type": "unlisten", "port": e.Unlisten})
				}
			}
		}
		g.last = now
	}
}

// invariant: the containers are asked every containersEach scans, and their
// last answer is merged into every scan between, so a container's port keeps
// its name and never flaps (its socket is the container engine's proxy, or
// none without the proxy).
func (s *Agent) scan(ctx context.Context, h Hello, held *[]ports.Announced, refresh bool, from, to int) []ports.Announced {
	var found []ports.Announced
	if s.o.Listeners != nil {
		if ls, err := s.o.Listeners.Read(ctx); err == nil {
			found = ports.Announce(ls)
		}
	}
	if refresh && s.o.Containers != nil {
		if mapped, err := s.o.Containers.Read(ctx); err == nil {
			*held = mapped
		}
	}
	found = ports.MergeAnnounced(found, *held)
	return ports.MarkQuiet(ports.FilterAnnounced(found, h.MinPort, h.Ignored), from, to)
}

func (s *Agent) writeFiles(h Hello) error {
	home := s.h.Env.Home()
	return errors.Join(s.writeKeys(path.Join(home, ".ssh", "authorized_keys"), h.Keys), s.writeEnv(h.Env), s.writeMac(h))
}

func (s *Agent) writeKeys(file string, keys []string) error {
	current, _ := s.h.FS.ReadFile(file)
	var kept []string
	inside := false
	for _, line := range strings.Split(strings.TrimRight(string(current), "\n"), "\n") {
		switch {
		case line == KeysBegin:
			inside = true
		case line == KeysEnd:
			inside = false
		case !inside && line != "":
			kept = append(kept, line)
		}
	}
	lines := append(append(append(kept, KeysBegin), keys...), KeysEnd)
	if err := s.h.FS.MkdirAll(path.Dir(file), 0o700); err != nil {
		return err
	}
	if err := s.h.FS.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return s.h.FS.Chmod(file, 0o600)
}

func Quote(v string) string { return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'" }

func (s *Agent) writeEnv(env map[string]string) error {
	names := make([]string, 0, len(env))
	for k := range env {
		names = append(names, k)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, k := range names {
		b.WriteString("export " + k + "=" + Quote(env[k]) + "\n")
	}
	file := s.o.PortsEnv
	if err := s.h.FS.MkdirAll(path.Dir(file), 0o700); err != nil {
		return err
	}
	return s.h.FS.WriteFile(file, []byte(b.String()), 0o644)
}

func (s *Agent) writeMac(h Hello) error {
	data, err := json.MarshalIndent(ports.VMLivesOn{Mac: h.Mac, VM: h.VM, PublicName: h.PublicName}, "", "  ")
	if err != nil {
		return err
	}
	if err := s.h.FS.MkdirAll(s.o.State, 0o700); err != nil {
		return err
	}
	return s.h.FS.WriteFile(path.Join(s.o.State, "guest.json"), append(data, '\n'), 0o600)
}

func (s *Agent) Placement() ports.VMLivesOn {
	var p ports.VMLivesOn
	data, err := s.h.FS.ReadFile(path.Join(s.o.State, "guest.json"))
	if err == nil {
		_ = json.Unmarshal(data, &p)
	}
	return p
}

func (s *Agent) listen(ctx context.Context, g *session, quit context.CancelFunc) local.Listener {
	file := s.SocketPath()
	if conn, err := s.h.Sockets.DialUnix(ctx, file, time.Second); err == nil {
		_, _ = conn.Write([]byte(`{"v":2,"id":"q","call":"` + callQuit + `"}` + "\n"))
		_, _ = bufio.NewReader(conn).ReadString('\n')
		_ = conn.Close()
	}
	_ = s.h.FS.Remove(file)
	if err := s.h.FS.MkdirAll(path.Dir(file), 0o700); err != nil {
		return nil
	}
	l, err := s.h.Sockets.ListenUnix(file)
	if err != nil {
		return nil
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go g.serve(ctx, conn, quit)
		}
	}()
	return l
}

// value: the calls the socket inside a VM serves, in station.api/v2 frames:
// the four a command here asks its Mac's engine, relayed over the guest
// channel, and the quit a newer guest sends the one it replaces.
const (
	callPorts      = "engine.ports"
	callReserve    = "engine.reserve"
	callRelease    = "engine.release"
	callAgentsHeld = "engine.agents-held"
	callQuit       = "guest.quit"
)

type request struct {
	V    int    `json:"v"`
	ID   string `json:"id"`
	Call string `json:"call"`
	In   struct {
		Port int `json:"guest_port"`
	} `json:"in"`
}

type fault struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func (g *session) serve(ctx context.Context, conn local.Conn, quit context.CancelFunc) {
	defer func() { _ = conn.Close() }()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var r request
		if json.Unmarshal(scanner.Bytes(), &r) != nil {
			continue
		}
		if r.Call == callQuit {
			_, _ = conn.Write([]byte(`{"v":2,"id":"` + r.ID + `","out":null}` + "\n"))
			quit()
			return
		}
		frame := map[string]any{"v": 2, "id": r.ID}
		if op, ok := map[string]string{callPorts: "ports", callReserve: "reserve", callRelease: "release", callAgentsHeld: "agents-held"}[r.Call]; !ok || r.V != 2 {
			frame["error"] = fault{Type: "refused", Code: "call.unknown", Message: "no call is named " + r.Call, Hint: "station ports"}
		} else if reply := g.relay(ctx, op, r.In.Port); reply.OK {
			frame["out"] = reply.Result
		} else {
			var f fault
			_ = json.Unmarshal(reply.Error, &f)
			f.Type = "failed"
			if strings.HasPrefix(f.Code, "unknown_") {
				f.Type = "refused"
			}
			frame["error"] = f
		}
		data, _ := json.Marshal(frame)
		if _, err := conn.Write(append(data, '\n')); err != nil {
			return
		}
	}
}

func (g *session) relay(ctx context.Context, op string, port int) message {
	g.mu.Lock()
	g.seq++
	id := "g" + strconv.Itoa(g.seq)
	ch := make(chan message, 1)
	g.pending[id] = ch
	g.mu.Unlock()
	g.send(map[string]any{"v": 1, "type": "request", "id": id, "op": op, "port": port})
	ctx, cancel := context.WithTimeout(ctx, relayBudget)
	defer cancel()
	select {
	case m := <-ch:
		return m
	case <-ctx.Done():
		return message{Error: json.RawMessage(`{"code":"internal","message":"the host did not answer","hint":"station services"}`)}
	}
}
