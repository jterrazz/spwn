package world

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/settings"

	"github.com/jterrazz/studio-station/src/machines"
)

type Event struct {
	Name string
	Args []string
}

type event struct {
	name  string
	args  []string
	apply func(w *World, args []string) error
}

// invariant: the closed list of world events (USE-CASES §0.2), each one
// played by the fakes.
func events() []event {
	return []event{
		{"reboot", []string{"<mac>"}, reboot},
		{"power-cut", []string{"<mac>"}, powerCut},
		{"unlock", []string{"<mac>"}, unlock},
		{"offline", []string{"<mac>"}, offline(true)},
		{"online", []string{"<mac>"}, offline(false)},
		{"engine-crash", []string{"<mac>"}, engineCrash},
		{"crash", []string{"<vm>"}, ended(machines.RunEnd{Reason: machines.Crashed, ExitCode: 1})},
		{"poweroff-inside", []string{"<vm>"}, ended(machines.RunEnd{Reason: machines.GuestPoweroff})},
		{"reboot-inside", []string{"<vm>"}, ended(machines.RunEnd{Reason: machines.GuestReboot})},
		{"console", []string{"<vm>", "<line>"}, writeConsole},
		{"listen", []string{"<machine>", "<bind>:<port>", "<process>"}, listen},
		{"unlisten", []string{"<machine>", "<port>"}, unlisten},
		{"container-run", []string{"<vm>", "<port>:<port>", "<container>", "[no-proxy]"}, containerRun},
		{"edit", []string{"<machine>", "<file>", "[<text>]"}, edit},
		{"unplug", nil, power(battery, func(p *Power) error { p.Source = "battery"; p.Charging = false; return nil })},
		{"plug", nil, power(battery, func(p *Power) error { p.Source = "ac"; return nil })},
		{"close-lid", nil, power(lid, func(p *Power) error { p.Lid = "closed"; return nil })},
		{"open-lid", nil, power(lid, func(p *Power) error { p.Lid = "open"; return nil })},
		{"will-sleep", []string{"<mac>"}, willSleep},
		{"thermal", []string{"<level>"}, thermal},
		{"battery", []string{"<percent>"}, percent},
		{"low-power", []string{"on|off"}, lowPower},
		{"advance", []string{"<duration>"}, advance},
		{"fault", []string{"<Contract>.<Method>", "[<n>]", "[times]"}, fault},
		{"hold", []string{"<Contract>.<Method>"}, hold},
		{"upgrade", []string{"<mac>", "<version>"}, upgrade},
		{"address", []string{"<vm>", "<ip>"}, address},
		{"slow", []string{"<machine>", "<duration>"}, slow},
		{"reads", []string{"<mac>", "<fact>", "<value>"}, reads},
		{"certificate", []string{"<mac>", "pending|issued"}, certificate},
		{"admin", []string{"on|off"}, admin},
		{"agent-prompt", []string{"<vm>", "<session>", "<prompt>", "[<folder>]"}, agentEvent(agents.Prompted, agentPrompt)},
		{"agent-step", []string{"<vm>", "<session>", "edit|read|run|web|delegate", "<path-or-command>", "[<output>]"}, agentUse},
		{"agent-notify", []string{"<vm>", "<session>", "<message>"}, agentEvent(agents.Notified, agentMessage)},
		{"agent-stop", []string{"<vm>", "<session>", "<message>"}, agentEvent(agents.Stopped, agentMessage)},
		{"agent-end", []string{"<vm>", "<session>", "[<reason>]"}, agentEvent(agents.Ended, agentEnd)},
	}
}

func Names() []string {
	out := make([]string, 0, len(events()))
	for _, e := range events() {
		out = append(out, e.name)
	}
	return out
}

func Parse(args []string) (Event, error) {
	if len(args) == 0 {
		return Event{}, fmt.Errorf("no world event: one of %s", strings.Join(Names(), ", "))
	}
	i := slices.IndexFunc(events(), func(e event) bool { return e.name == args[0] })
	if i < 0 {
		return Event{}, fmt.Errorf("unknown world event %q: one of %s", args[0], strings.Join(Names(), ", "))
	}
	e := events()[i]
	need := 0
	for _, a := range e.args {
		if !strings.HasPrefix(a, "[") {
			need++
		}
	}
	if len(args)-1 < need || len(args)-1 > len(e.args) {
		return Event{}, fmt.Errorf("world %s takes %s", e.name, strings.Join(e.args, " "))
	}
	return Event{Name: e.name, Args: args[1:]}, nil
}

func (w *World) Apply(ev Event) error {
	i := slices.IndexFunc(events(), func(e event) bool { return e.name == ev.Name })
	if i < 0 {
		return fmt.Errorf("unknown world event %q", ev.Name)
	}
	return events()[i].apply(w, ev.Args)
}

func (w *World) known(mac string) (*Mac, error) {
	if _, ok := w.Macs[mac]; !ok {
		return nil, fmt.Errorf("no Mac %q in this world", mac)
	}
	return w.Mac(mac), nil
}

func (w *World) vm(name string) (*VM, error) {
	mac := w.Holder(name)
	if mac == "" {
		return nil, fmt.Errorf("no VM %q in this world", name)
	}
	return w.Macs[mac].VMs[name], nil
}

// invariant: a Mac that goes down ends every run on it with nobody to read
// how; its engine starts again at login, even one the owner stopped, and
// finds no run to adopt.
func reboot(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	for _, v := range m.VMs {
		v.Down(w.Now, nil)
	}
	if m.Engine != nil {
		m.Engine.Stopped, m.Engine.StartedAt = false, w.Now
	}
	if m.Power != nil {
		m.Power.Holds, m.Power.Clamshell = nil, false
	}
	return nil
}

func powerCut(w *World, args []string) error {
	if err := reboot(w, args); err != nil {
		return err
	}
	w.Macs[args[0]].Locked = true
	return nil
}

func unlock(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	m.Locked = false
	return nil
}

// invariant: macOS tells a Mac's engine it is going to sleep; the Mac
// still answers until it goes offline.
func willSleep(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	if m.Power == nil {
		return fmt.Errorf("%s has no power management in this world", args[0])
	}
	m.Power.Sleeping = true
	return nil
}

func offline(off bool) func(w *World, args []string) error {
	return func(w *World, args []string) error {
		m, err := w.known(args[0])
		if err != nil {
			return err
		}
		m.Offline = off
		return nil
	}
}

// invariant: the engine's job manager starts a crashed engine again: the
// runs it supervised outlive it and are adopted.
func engineCrash(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	if m.Engine == nil {
		return fmt.Errorf("no engine runs on %s", args[0])
	}
	// invariant: the assertions an engine took end with its process; the
	// clamshell it set stays until its successor resets it.
	if m.Power != nil {
		m.Power.Holds = nil
	}
	return nil
}

func ended(end machines.RunEnd) func(w *World, args []string) error {
	return func(w *World, args []string) error {
		v, err := w.vm(args[0])
		if err != nil {
			return err
		}
		if !v.Running() {
			return fmt.Errorf("%s does not run", args[0])
		}
		v.Down(w.Now, &end)
		return nil
	}
}

func writeConsole(w *World, args []string) error {
	v, err := w.vm(args[0])
	if err != nil {
		return err
	}
	v.Console = append(v.Console, args[1])
	return nil
}

// invariant: a file the owner edits by hand is the text the story gives it,
// read by the store that owns it at its next read.
func edit(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	if m.Files == nil {
		m.Files = map[string]string{}
	}
	text := ""
	if len(args) > 2 {
		text = args[2]
	}
	m.Files[args[1]] = text
	return nil
}

func advance(w *World, args []string) error {
	d, err := time.ParseDuration(args[0])
	if err != nil {
		return fmt.Errorf("world advance: %w", err)
	}
	w.Now = w.Now.Add(d)
	return nil
}

// invariant: a slow machine's disk answers each size that late, in real
// time: a walk the owner watches fill.
func slow(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	d, err := time.ParseDuration(args[1])
	if err != nil {
		return fmt.Errorf("world slow: %w", err)
	}
	if m.Disk == nil {
		m.Disk = &Disk{}
	}
	m.Disk.Late = d
	return nil
}

// invariant: a Mac whose fact changes outside station (the owner turns the
// firewall off): a value off or never reads absent, one with updates
// pending degraded, any other in place.
func reads(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	m.Reads(catalog.Fact(args[1]), args[2])
	switch value := args[2]; {
	case value == "off", value == "never":
		m.Facts[catalog.Fact(args[1])] = settings.Fact{State: catalog.StateAbsent, Value: value}
	case strings.HasSuffix(value, " pending"):
		m.Facts[catalog.Fact(args[1])] = settings.Fact{State: catalog.StateDegraded, Value: value}
	}
	return nil
}

func upgrade(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	if m.Engine == nil {
		m.Engine = &Engine{}
	}
	m.Engine.Version = args[1]
	if m.Station != "" {
		m.Station, m.Lacks = args[1], nil
	}
	return nil
}

func address(w *World, args []string) error {
	v, err := w.vm(args[0])
	if err != nil {
		return err
	}
	v.Address = args[1]
	return nil
}

// invariant: what listens in a VM is its record's on its Mac, also when the
// world holds the VM's own slice for a binary run inside it.
func (w *World) listening(machine string) (*[]ports.Listener, error) {
	if v, err := w.vm(machine); err == nil {
		return &v.Listening, nil
	}
	if m, ok := w.Macs[machine]; ok {
		return &m.Listening, nil
	}
	return nil, fmt.Errorf("no machine %q in this world", machine)
}

func (w *World) Listening(machine string) []ports.Listener {
	at, err := w.listening(machine)
	if err != nil {
		return nil
	}
	return *at
}

func (w *World) Containers(vm string) []Container {
	v, err := w.vm(vm)
	if err != nil {
		return nil
	}
	return v.Containers
}

func listen(w *World, args []string) error {
	at, err := w.listening(args[0])
	if err != nil {
		return err
	}
	i := strings.LastIndex(args[1], ":")
	port, perr := strconv.Atoi(args[1][i+1:])
	if i < 0 || perr != nil {
		return fmt.Errorf("world listen: %q is not <bind>:<port>", args[1])
	}
	bind := strings.Trim(args[1][:i], "[]")
	*at = append(*at, ports.Listener{Port: port, Bind: bind, Process: args[2], PID: 2000 + len(*at)})
	return nil
}

func unlisten(w *World, args []string) error {
	at, err := w.listening(args[0])
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("world unlisten: %q is not a port", args[1])
	}
	*at = slices.DeleteFunc(*at, func(l ports.Listener) bool { return l.Port == port })
	return nil
}

// invariant: a container publishes its port on the VM's every interface;
// its engine's userland proxy listens on it too, unless the VM runs without
// it.
func containerRun(w *World, args []string) error {
	v, err := w.vm(args[0])
	if err != nil {
		return err
	}
	shared, _, ok := strings.Cut(args[1], ":")
	port, perr := strconv.Atoi(shared)
	if !ok || perr != nil {
		return fmt.Errorf("world container-run: %q is not <port>:<port>", args[1])
	}
	if len(args) == 4 && args[3] != "no-proxy" {
		return fmt.Errorf("world container-run: %q is not no-proxy", args[3])
	}
	v.Containers = append(v.Containers, Container{Name: args[2], Shared: []int{port}})
	if len(args) < 4 {
		v.Listening = append(v.Listening, ports.Listener{Port: port, Bind: "0.0.0.0", Process: "proxy", PID: 2000 + len(v.Listening)})
	}
	return nil
}

// value: the capability a power event needs of this Mac.
type need int

const (
	anyPower need = iota
	battery
	lid
)

// invariant: a power event moves the reading of the Mac the binary runs on,
// and is refused on one that has not the capability it moves.
func power(needs need, move func(p *Power) error) func(w *World, args []string) error {
	return func(w *World, _ []string) error {
		m, err := w.known(w.ThisMac)
		if err != nil {
			return err
		}
		p := m.Power
		switch {
		case p == nil:
			return fmt.Errorf("%s has no power management", w.ThisMac)
		case needs == battery && !p.HasBattery:
			return fmt.Errorf("%s has no battery", w.ThisMac)
		case needs == lid && p.Lid == "":
			return fmt.Errorf("%s has no lid", w.ThisMac)
		}
		return move(p)
	}
}

func thermal(w *World, args []string) error {
	return power(anyPower, func(p *Power) error {
		for n := 0; n <= awake.ThermalEnds+1; n++ {
			if awake.Level(n) == args[0] || strconv.Itoa(n) == args[0] {
				p.Thermal = n
				return nil
			}
		}
		return fmt.Errorf("world thermal: %q is no thermal level", args[0])
	})(w, args)
}

func percent(w *World, args []string) error {
	return power(battery, func(p *Power) error {
		n, err := strconv.Atoi(strings.TrimSuffix(args[0], "%"))
		if err != nil || n < 0 || n > 100 {
			return fmt.Errorf("world battery: %q is no percent", args[0])
		}
		p.Percent = n
		return nil
	})(w, args)
}

func lowPower(w *World, args []string) error {
	return power(anyPower, func(p *Power) error {
		switch args[0] {
		case "on", "off":
			p.LowPower = args[0] == "on"
			return nil
		}
		return fmt.Errorf("world low-power takes on or off, not %q", args[0])
	})(w, args)
}

func admin(w *World, args []string) error {
	switch args[0] {
	case "on", "off":
		w.Admin = args[0] == "on"
		return nil
	}
	return fmt.Errorf("world admin takes on or off, not %q", args[0])
}

func hold(w *World, args []string) error {
	if contract, method, ok := strings.Cut(args[0], "."); !ok || contract == "" || method == "" {
		return fmt.Errorf("world hold: %q is not <Contract>.<Method>", args[0])
	}
	w.Script = append(w.Script, Fault{Call: args[0], Hold: true})
	return nil
}

func fault(w *World, args []string) error {
	call := args[0]
	if contract, method, ok := strings.Cut(call, "."); !ok || contract == "" || method == "" {
		return fmt.Errorf("world fault: %q is not <Contract>.<Method>", call)
	}
	f := Fault{Call: call}
	if len(args) > 1 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || (len(args) == 3 && args[2] != "times" && args[2] != "time") {
			return fmt.Errorf("world fault %s takes [<n> times], not %q", call, strings.Join(args[1:], " "))
		}
		f.Times = n
	}
	w.Script = append(w.Script, f)
	return nil
}

func certificate(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	if m.Tailnet == nil {
		return fmt.Errorf("world certificate: %s is not on the tailnet", args[0])
	}
	switch args[1] {
	case "pending", "issued":
		m.Tailnet.CertificatePending = args[1] == "pending"
		return nil
	}
	return fmt.Errorf("world certificate: %q is not pending or issued", args[1])
}
