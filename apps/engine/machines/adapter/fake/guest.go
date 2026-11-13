package fake

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

var (
	ErrNoGuest      = errors.New("no guest answers at this address")
	ErrClosed       = errors.New("the guest connection is closed")
	ErrNotListening = errors.New("nothing listens on this guest port")
	ErrRefused      = errors.New("the guest connection is refused, as the world scripted it")
)

type Answer struct {
	Code           int
	Stdout, Stderr string
}

type File struct {
	Data []byte
	Mode fs.FileMode
}

type VM struct {
	mu      sync.Mutex
	answers map[string]Answer
	// value: the lines that power the VM off once answered.
	poweroffs map[string]bool
	received  map[string][]byte
	files     map[string]File
	listeners map[int]string
}

func NewVM() *VM {
	return &VM{answers: map[string]Answer{}, poweroffs: map[string]bool{}, received: map[string][]byte{}, files: map[string]File{}, listeners: map[int]string{}}
}

func (v *VM) Program(line string, a Answer) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.answers[line] = a
}

func (v *VM) Serve(port int, greeting string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.listeners[port] = greeting
}

func (v *VM) Received(line string) []byte {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.received[line]
}

func (v *VM) File(path string) (File, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	f, ok := v.files[path]
	return f, ok
}

type Guests struct {
	// invariant: the guests of the world's VMs: a VM whose slice has a guest
	// answers what it says, one with none is not reached.
	w   *world.World
	mu  sync.Mutex
	vms map[string]*VM
	// value: the line station's own session runs on in a guest.
	session string
}

func (g *Guests) Serving(session string) *Guests {
	g.session = session
	return g
}

func GuestsOf(w *world.World) *Guests { return &Guests{w: w, vms: map[string]*VM{}} }

func NewGuests() *Guests { return GuestsOf(&world.World{ThisMac: "this-mac"}) }

func (g *Guests) Add(vm string) *VM {
	g.w.Lock()
	if g.w.Holder(vm) == "" {
		g.w.Mac(g.w.ThisMac).VMs[vm] = &world.VM{}
	}
	g.w.Macs[g.w.Holder(vm)].VMs[vm].Guest = &world.Guest{}
	g.w.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	v := NewVM()
	g.vms[vm] = v
	return v
}

func (g *Guests) answering(vm string) (*world.Guest, bool) {
	g.w.Lock()
	defer g.w.Unlock()
	mac := g.w.Holder(vm)
	if mac == "" || g.w.Macs[mac].VMs[vm].Guest == nil {
		return nil, false
	}
	return g.w.Macs[mac].VMs[vm].Guest, true
}

func (g *Guests) Dial(_ context.Context, t contract.GuestTarget) (contract.Guest, error) {
	g.w.Lock()
	refused := g.w.Fails("Guest.Dial")
	g.w.Unlock()
	if refused {
		return nil, ErrRefused
	}
	wg, ok := g.answering(t.VM)
	if !ok {
		return nil, ErrNoGuest
	}
	answers := wg.Answers
	g.mu.Lock()
	v, ok := g.vms[t.VM]
	if !ok {
		v = NewVM()
		for line, a := range answers {
			v.answers[line] = Answer{Code: a.Code, Stdout: a.Stdout, Stderr: a.Stderr}
			v.poweroffs[line] = a.Poweroff
		}
		g.vms[t.VM] = v
	}
	g.mu.Unlock()
	serving := g.session
	if wg.Replaced {
		serving = ""
	}
	return &guest{vm: v, name: t.VM, w: g.w, serving: serving, fresh: wg.Fresh, done: make(chan struct{})}, nil
}

type guest struct {
	vm   *VM
	name string
	w    *world.World
	// value: the line of station's own session, which this guest runs as its
	// image installs it; empty when its tooling replaced station's guest.
	serving string
	// value: a guest made from station's image: a line no answer scripts
	// succeeds, as the image's own commands do; `true` succeeds on any guest.
	fresh bool
	done  chan struct{}
	once  sync.Once
}

// invariant: station's session in the fake guest says it is ready, reports
// what listens in its VM of the world, as the guest's first scan does, takes
// whatever the engine writes, and lasts until the connection closes.
func (c *guest) serve(ctx context.Context, cmd contract.GuestCommand) (contract.GuestResult, error) {
	if cmd.Stdin != nil {
		go func() {
			buf := make([]byte, 4096)
			for {
				if _, err := cmd.Stdin.Read(buf); err != nil {
					return
				}
			}
		}()
	}
	if cmd.Stdout != nil {
		if _, err := cmd.Stdout.Write(append(append([]byte(`{"v":1,"type":"ready"}`+"\n"), c.snapshot()...), c.agents()...)); err != nil {
			return contract.GuestResult{}, err
		}
	}
	select {
	case <-ctx.Done():
	case <-c.done:
	}
	return contract.GuestResult{}, nil
}

// invariant: a guest reports a port once: a container's published port is
// named by its container, over the proxy its engine may run on it.
func (c *guest) snapshot() []byte {
	c.w.Lock()
	found := c.w.Reported(c.name)
	c.w.Unlock()
	line, err := json.Marshal(map[string]any{"v": 1, "type": "snapshot", "ports": found})
	if err != nil {
		return nil
	}
	return append(line, '\n')
}

// invariant: a guest reports its VM's journal as it is, as the guest's
// first look at it does: the sessions not past their keep, by name.
func (c *guest) agents() []byte {
	c.w.Lock()
	kept := []any{}
	if mac := c.w.Holder(c.name); mac != "" {
		if held := c.w.Macs[mac].VMs[c.name].Agents; held != nil {
			for _, name := range slices.Sorted(maps.Keys(held.Journal)) {
				if !held.Journal[name].Expired(c.w.Now) {
					kept = append(kept, held.Journal[name])
				}
			}
		}
	}
	c.w.Unlock()
	line, err := json.Marshal(map[string]any{"v": 1, "type": "agents", "agents": kept})
	if err != nil {
		return nil
	}
	return append(line, '\n')
}

var ErrGuestBroke = errors.New("the guest connection broke, as the world scripted it")

func (v *VM) answer(line string) (a Answer, poweroff, ok bool) {
	if a, ok := v.answers[line]; ok {
		return a, v.poweroffs[line], true
	}
	for pattern, a := range v.answers {
		if prefix, wild := strings.CutSuffix(pattern, "*"); wild && strings.HasPrefix(line, prefix) {
			return a, v.poweroffs[pattern], true
		}
	}
	return Answer{}, false, false
}

// invariant: a scripted Guest.Exec fault breaks a command the guest answers;
// a line it does not know spends no fault.
func (c *guest) fails() bool {
	c.w.Lock()
	defer c.w.Unlock()
	return c.w.Fails("Guest.Exec")
}

// invariant: a guest that powers its VM off ends the VM's run as a guest
// poweroff, and the connection with it.
func (c *guest) poweroff() {
	c.w.Lock()
	if mac := c.w.Holder(c.name); mac != "" {
		if v := c.w.Macs[mac].VMs[c.name]; v.Running() {
			v.PID, v.Ended = 0, &machines.RunEnd{Reason: machines.GuestPoweroff}
		}
	}
	c.w.Unlock()
	_ = c.Close()
}

func (c *guest) closed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *guest) Exec(ctx context.Context, cmd contract.GuestCommand) (contract.GuestResult, error) {
	if c.closed() {
		return contract.GuestResult{}, ErrClosed
	}
	if cmd.Line == c.serving && c.serving != "" {
		return c.serve(ctx, cmd)
	}
	if cmd.Line != "true" {
		c.ran(cmd.Line)
	}
	var in []byte
	if cmd.Stdin != nil {
		var err error
		if in, err = io.ReadAll(cmd.Stdin); err != nil {
			return contract.GuestResult{}, err
		}
	}
	c.vm.mu.Lock()
	a, poweroff, ok := c.vm.answer(cmd.Line)
	c.vm.mu.Unlock()
	switch {
	case !ok && (c.fresh || cmd.Line == "true"):
	case !ok:
		a = Answer{Code: 127, Stderr: cmd.Line + ": not found\n"}
	case c.fails():
		return contract.GuestResult{}, ErrGuestBroke
	default:
		c.vm.mu.Lock()
		c.vm.received[cmd.Line] = in
		c.vm.mu.Unlock()
	}
	res := contract.GuestResult{Code: a.Code, Stdout: a.Stdout, Stderr: a.Stderr}
	if cmd.Stdout != nil {
		if _, err := cmd.Stdout.Write([]byte(a.Stdout)); err != nil {
			return contract.GuestResult{}, err
		}
		res.Stdout = ""
	}
	if cmd.Stderr != nil {
		if _, err := cmd.Stderr.Write([]byte(a.Stderr)); err != nil {
			return contract.GuestResult{}, err
		}
		res.Stderr = ""
	}
	if poweroff {
		c.poweroff()
	}
	return res, nil
}

func (c *guest) ran(line string) {
	c.w.Lock()
	defer c.w.Unlock()
	if mac := c.w.Holder(c.name); mac != "" {
		if g := c.w.Macs[mac].VMs[c.name].Guest; g != nil {
			g.Ran = append(g.Ran, line)
		}
	}
}

func (c *guest) Forward(_ context.Context, _ string, port int) (io.ReadWriteCloser, error) {
	if c.closed() {
		return nil, ErrClosed
	}
	c.vm.mu.Lock()
	greeting, ok := c.vm.listeners[port]
	c.vm.mu.Unlock()
	if !ok {
		greeting, ok = c.answers(port)
	}
	if !ok {
		return nil, errors.Join(ErrNotListening, errors.New("port "+strconv.Itoa(port)))
	}
	return &stream{Reader: strings.NewReader(greeting + "\n")}, nil
}

// value: what a program that is no web server answers first, by its name.
func speaksFirst(process string) (string, bool) {
	switch process {
	case "redis-server":
		return "-ERR unknown command 'HEAD'", true
	case "sshd":
		return "SSH-2.0-OpenSSH_9.6", true
	}
	return "", false
}

// invariant: a port the world says listens in the VM answers as a web
// server does, the dev server a workspace runs, unless its program is one
// that speaks its own protocol first; a port a story scripts answers its
// own greeting.
func (c *guest) answers(port int) (string, bool) {
	c.w.Lock()
	defer c.w.Unlock()
	for _, l := range c.w.Listening(c.name) {
		if l.Port == port {
			if g, ok := speaksFirst(l.Process); ok {
				return g, true
			}
			return "HTTP/1.1 200 OK", true
		}
	}
	if slices.ContainsFunc(c.w.Containers(c.name), func(k world.Container) bool { return slices.Contains(k.Shared, port) }) {
		return "HTTP/1.1 200 OK", true
	}
	return "", false
}

type stream struct{ *strings.Reader }

func (s *stream) Write(p []byte) (int, error) { return len(p), nil }
func (s *stream) Close() error                { return nil }

func (c *guest) Put(_ context.Context, path string, data []byte, mode fs.FileMode) error {
	if c.closed() {
		return ErrClosed
	}
	c.vm.mu.Lock()
	defer c.vm.mu.Unlock()
	c.vm.files[path] = File{Data: append([]byte(nil), data...), Mode: mode}
	return nil
}

func (c *guest) Close() error {
	c.once.Do(func() { close(c.done) })
	return nil
}

func (c *guest) Done() <-chan struct{} { return c.done }
