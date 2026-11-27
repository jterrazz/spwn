package world

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/settings"

	"github.com/jterrazz/studio-station/apps/engine/machines"
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
		{"knock", []string{"<mac>", "<from-mac>", "[not-the-engine]"}, knock},
		{"engine-log", []string{"<mac>", "<line>"}, writeEngineLog},
		{"crash", []string{"<vm>"}, ended(machines.RunEnd{Reason: machines.Crashed, ExitCode: 1})},
		{"poweroff-inside", []string{"<vm>"}, ended(machines.RunEnd{Reason: machines.GuestPoweroff})},
		{"reboot-inside", []string{"<vm>"}, ended(machines.RunEnd{Reason: machines.GuestReboot})},
		{"console", []string{"<vm>", "<line>"}, writeConsole},
		{"hypervisor", []string{"<vm>", "<line>"}, writeHypervisor},
		{"listen", []string{"<machine>", "<bind>:<port>", "<process>"}, listen},
		{"unlisten", []string{"<machine>", "<port>"}, unlisten},
		{"container-run", []string{"<vm>", "<port>:<port>", "<container>", "[no-proxy]"}, containerRun},
		{"edit", []string{"<machine>", "<file>", "[<text>]"}, edit},
		{"building", []string{"<image>", "<step>"}, building},
		{"unplug", nil, power(battery, func(p *Power) error { p.Source = "battery"; p.Charging = false; return nil })},
		{"plug", nil, power(battery, func(p *Power) error { p.Source = "ac"; return nil })},
		{"close-lid", nil, power(lid, func(p *Power) error { p.Lid = "closed"; return nil })},
		{"open-lid", nil, power(lid, func(p *Power) error { p.Lid = "open"; return nil })},
		{"will-sleep", []string{"<mac>"}, willSleep},
		{"thermal", []string{"<level>"}, thermal},
		{"battery", []string{"<percent>"}, percent},
		{"low-power", []string{"on|off"}, lowPower},
		{"advance", []string{"<duration>"}, advance},
		{"upstream-login", []string{"<user>", "<password>"}, upstreamLogin},
		{"lockdown-cut", []string{"before-lock|before-key|after-key|never-ends|drops-after-print", "[<n>]"}, lockdownCut},
		{"fault", []string{"<Contract>.<Method>", "[<n>]", "[times]"}, fault},
		{"panic", []string{"<Contract>.<Method>", "[<n>]", "[times]"}, panicking},
		{"hold", []string{"<Contract>.<Method>"}, hold},
		{"unhold", []string{"[<mac>]", "[<vm>]"}, unhold},
		{"upgrade", []string{"<machine>", "<version>"}, upgrade},
		{"address", []string{"<vm>", "<ip>"}, address},
		{"slow", []string{"<machine>", "<duration>"}, slow},
		{"reads", []string{"<mac>", "<fact>", "<value>"}, reads},
		{"unopened", []string{"<mac>", "<app>"}, unopened},
		{"certificate", []string{"<mac>", "pending|issued"}, certificate},
		{"local-network", []string{"<mac>", "approved|not-approved"}, localNetwork},
		{"agent", []string{"<mac>", "served|held", "[<pid>]", "[<executable>]", "[<label>]"}, agentSocket},
		{"admin", []string{"on|off"}, admin},
		{"release", []string{"<tag>", "[no-asset]"}, releaseTag},
		{"project", []string{"<root>", "[error|warning|info|drift|render|edit|link|fifo|readonly]", "[<agent>|<path>]", "[<path>|<message>|<content>|<target>|off]", "[<content>|hint:]", "[<hint>]"}, declareProject},
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
	m.Processes = nil
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

func knock(w *World, args []string) error {
	// invariant: a knock changes nothing of the world: the second Mac's
	// device key knocks at the first's door, and what the door says is the
	// door's own line, played beside the world by its run.
	if len(args) == 3 && args[2] != "not-the-engine" {
		return fmt.Errorf("world knock takes [not-the-engine], not %q", args[2])
	}
	for _, mac := range args[:2] {
		if _, err := w.known(mac); err != nil {
			return err
		}
	}
	return nil
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
	// invariant: the assertions an engine took end with its process, and
	// the device book's head it held; the clamshell it set stays until its
	// successor resets it.
	if m.Power != nil {
		m.Power.Holds = nil
	}
	m.Engine.Book = nil
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

func writeEngineLog(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	m.EngineLog = append(m.EngineLog, args[1])
	return nil
}

func writeHypervisor(w *World, args []string) error {
	v, err := w.vm(args[0])
	if err != nil {
		return err
	}
	v.Hypervisor = append(v.Hypervisor, args[1])
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

func upstreamLogin(w *World, args []string) error {
	w.UpstreamLogin = &ImageLogin{User: args[0], Password: args[1]}
	return nil
}

func lockdownCut(w *World, args []string) error {
	if !slices.Contains([]string{CutBeforeLock, CutBeforeKey, CutAfterKey, CutNeverEnds, CutDropsAfterPrint}, args[0]) {
		return fmt.Errorf("lockdown-cut takes before-lock, before-key, after-key, never-ends or drops-after-print, not %q", args[0])
	}
	n := 1
	if len(args) > 1 {
		var err error
		if n, err = strconv.Atoi(args[1]); err != nil || n < 1 {
			return fmt.Errorf("lockdown-cut takes a count of at least 1, not %q", args[1])
		}
	}
	for range n {
		w.LockdownCuts = append(w.LockdownCuts, args[0])
	}
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

func unopened(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	m.Unopened = append(m.Unopened, args[1])
	return nil
}

// invariant: a VM's station is the one inside it, which its guest serves;
// a Mac's is its engine's, and its station's once the world names one.
func upgrade(w *World, args []string) error {
	if slice, ok := w.Macs[args[0]]; ok && w.Holder(args[0]) != "" {
		slice.Station = args[1]
		return nil
	}
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	if m.Engine == nil {
		m.Engine = &Engine{}
	}
	m.Engine.Version, m.Engine.Lacks = args[1], nil
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

func (w *World) Reported(machine string) []ports.Announced {
	// invariant: a guest reports a port once: a container's published port is
	// named by its container, over the proxy its engine may run on it; a
	// socket's is marked quiet as a Linux guest marks it. The caller holds the
	// world's lock.
	listening, containers := w.Listening(machine), w.Containers(machine)
	found := make([]ports.Announced, 0, len(listening))
	for _, l := range listening {
		if !slices.ContainsFunc(containers, func(k Container) bool { return slices.Contains(k.Shared, l.Port) }) {
			found = append(found, ports.Announced{Port: l.Port, IP: l.Bind, Process: l.Process, Source: ports.BySocket})
		}
	}
	found = ports.MarkQuiet(found, ports.LinuxEphemeralFrom, ports.LinuxEphemeralTo)
	for _, k := range containers {
		for _, port := range k.Shared {
			found = append(found, ports.Announced{Port: port, IP: "0.0.0.0", Process: k.Name, Source: ports.ByContainer})
		}
	}
	return found
}

func (w *World) Containers(vm string) []Container {
	v, err := w.vm(vm)
	if err != nil {
		return nil
	}
	return v.Containers
}

func building(w *World, args []string) error {
	// invariant: a build of the image, at the step, runs on This Mac in a
	// process of its own, and marks it as the real build marks its step; a
	// build of that image already marked is replaced.
	step := images.BuildStep(args[1])
	if !slices.Contains(images.BuildSteps(), step) {
		return fmt.Errorf("world building: %q is not a step of a build: one of %s", args[1], strings.Join(step.Enum(), ", "))
	}
	m := w.Mac(w.ThisMac)
	owner := machines.Process{PID: 4000 + len(m.Processes), Start: strconv.FormatInt(w.Now.Unix(), 10) + ".000000"}
	m.Processes = append(m.Processes, owner)
	m.Building = append(slices.DeleteFunc(m.Building, func(o images.Making) bool { return o.Image == args[0] }),
		images.Making{Image: args[0], Version: w.Now.Format("2006.01.02"), Step: step, Owner: &owner})
	return nil
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

// invariant: an unhold with nothing held changes nothing and fails nothing,
// so a recorder may run it after every gesture.
func unhold(w *World, args []string) error {
	switch len(args) {
	case 0:
		w.Unhold("", "")
		return nil
	case 2:
		if _, err := w.known(args[0]); err != nil {
			return err
		}
		w.Unhold(args[0], args[1])
		return nil
	}
	return fmt.Errorf("world unhold takes <mac> <vm>, or nothing, not %q", strings.Join(args, " "))
}

func fault(w *World, args []string) error { return scripted(w, "fault", args, false) }

func panicking(w *World, args []string) error { return scripted(w, "panic", args, true) }

func scripted(w *World, event string, args []string, panics bool) error {
	call := args[0]
	if contract, method, ok := strings.Cut(call, "."); !ok || contract == "" || method == "" {
		return fmt.Errorf("world %s: %q is not <Contract>.<Method>", event, call)
	}
	f := Fault{Call: call, Panic: panics}
	if len(args) > 1 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || (len(args) == 3 && args[2] != "times" && args[2] != "time") {
			return fmt.Errorf("world %s %s takes [<n> times], not %q", event, call, strings.Join(args[1:], " "))
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

func localNetwork(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	switch v := machines.LocalNetwork(args[1]); v {
	case machines.LocalNetworkApproved, machines.LocalNetworkNotApproved:
		m.LocalNetwork = v
		return nil
	case machines.LocalNetworkChecking:
	}
	return fmt.Errorf("world local-network: %q is not approved or not-approved", args[1])
}

func agentSocket(w *World, args []string) error {
	m, err := w.known(args[0])
	if err != nil {
		return err
	}
	switch {
	case len(args) == 2 && args[1] == "served":
		m.AgentHeld = nil
		return nil
	case len(args) >= 3 && args[1] == "held":
		pid, convErr := strconv.Atoi(args[2])
		if convErr != nil {
			return fmt.Errorf("world agent: %q is not a pid", args[2])
		}
		held := AgentHolder{PID: pid}
		if len(args) > 3 {
			held.Executable = args[3]
		}
		if len(args) > 4 {
			held.Label = args[4]
		}
		m.AgentHeld = &held
		return nil
	}
	return fmt.Errorf("world agent: %q is not served or held <pid>", strings.Join(args[1:], " "))
}

// invariant: the release made last is the latest; one made no-asset
// carries the Linux binaries alone, as between its making and the attach
// of its darwin assets.
func releaseTag(w *World, args []string) error {
	r := Release{Tag: args[0], URL: "https://github.com/" + catalog.StationRepo + "/releases/tag/" + args[0]}
	switch {
	case len(args) == 1:
		r.Assets = []string{"station-linux-amd64", "station-linux-arm64", "station-darwin-amd64", "station-darwin-arm64"}
	case args[1] == "no-asset":
		r.Assets = []string{"station-linux-amd64", "station-linux-arm64"}
	default:
		return fmt.Errorf("world release %s takes [no-asset], not %q", args[0], args[1])
	}
	w.Releases = append(w.Releases, r)
	return nil
}

// invariant: a project of This Mac at an absolute root, declared once and
// then given, one event each, an issue its plan finds or a row of its drift.
func declareProject(w *World, args []string) error {
	m, err := w.known(w.ThisMac)
	if err != nil {
		return err
	}
	root := args[0]
	if !filepath.IsAbs(root) {
		return fmt.Errorf("project takes an absolute root, not %q", root)
	}
	s := m.Store()
	if s.Projects == nil {
		s.Projects = map[string]project.State{}
	}
	kept, ok := s.Projects[root]
	if !ok {
		kept.Project = project.Project{Name: filepath.Base(root), Root: root, Manifest: filepath.Join(root, project.ManifestName)}
	}
	if len(args) > 1 {
		if len(args) < 3 || len(args) < 4 && args[1] != "edit" && args[1] != "fifo" && args[1] != "readonly" {
			return fmt.Errorf("project %s takes a level, a path and a message", root)
		}
		said, hint := args[3:], []string(nil)
		if i := slices.Index(said, "hint:"); i >= 0 {
			said, hint = said[:i], said[i+1:]
		}
		issue := project.Issue{Level: project.Level(args[1]), Path: args[2], Message: strings.Join(said, " "), Hint: strings.Join(hint, " "), Source: project.SourceManifest}
		switch {
		case args[1] == "render":
			declareRender(&kept, args[2], args[3], args[4:])
		case args[1] == "edit":
			editRunFolder(&kept, args[2], args[3:])
		case args[1] == "link":
			linkRunFolder(&kept, args[2], args[3])
		case args[1] == "fifo":
			kept.Irregular = append(kept.Irregular, args[2])
		case args[1] == "readonly":
			kept.ReadOnly = slices.DeleteFunc(kept.ReadOnly, func(at string) bool { return at == args[2] })
			if len(args) < 4 || args[3] != "off" {
				kept.ReadOnly = append(kept.ReadOnly, args[2])
			}
		case args[1] == "drift":
			issue.Level, issue.Source = project.Error, project.SourceLockfile
			kept.Drift = append(kept.Drift, issue)
		case slices.Contains(project.Level("").Enum(), args[1]):
			kept.Issues = append(kept.Issues, issue)
		default:
			return fmt.Errorf("project takes error, warning, info, drift, render, edit, link, fifo or readonly, not %q", args[1])
		}
	}
	s.Projects[root] = kept
	return nil
}

// invariant: render <agent> <path> <content> is what the agent's harness
// renders at that path of its run folder; with no content, it renders
// nothing there any more.
func declareRender(kept *project.State, agent, path string, content []string) {
	if kept.Rendered == nil {
		kept.Rendered = map[string][]project.File{}
	}
	files := slices.DeleteFunc(kept.Rendered[agent], func(f project.File) bool { return f.Path == path })
	if len(content) > 0 {
		files = append(files, project.File{Path: path, Data: []byte(strings.Join(content, " ") + "\n"), Mode: 0o644})
	}
	kept.Rendered[agent] = files
	if !slices.Contains(kept.Project.Agents, agent) {
		kept.Project.Agents = append(kept.Project.Agents, agent)
		slices.Sort(kept.Project.Agents)
	}
}

// invariant: edit <path> <content> is the owner writing that file by hand,
// a path from the project's root; with no content, deleting it.
func editRunFolder(kept *project.State, path string, content []string) {
	if kept.Disk == nil {
		kept.Disk = map[string]project.File{}
	}
	if len(content) == 0 {
		delete(kept.Disk, path)
		return
	}
	kept.Disk[path] = project.File{Path: path, Data: []byte(strings.Join(content, " ") + "\n"), Mode: 0o644}
}

// invariant: link <path> <target> is the owner, or a repository cloned,
// putting a link at that path from the project's root: a name of the
// declaration, a folder of a run folder's path or the record, which a
// project refuses to follow. fifo <path> puts a FIFO there. readonly
// <folder> takes the folder's write bit away, so no name in it is made,
// replaced or removed; readonly <folder> off gives it back.
func linkRunFolder(kept *project.State, path, target string) {
	if kept.Disk == nil {
		kept.Disk = map[string]project.File{}
	}
	kept.Disk[path] = project.File{Path: path, Data: []byte(target), Link: true}
}
