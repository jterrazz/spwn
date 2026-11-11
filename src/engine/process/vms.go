package process

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"github.com/jterrazz/studio-station/src/agents"
	"maps"
	"net/netip"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/ports"

	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
)

const (
	maxBackoff = 30 * time.Second
)

type vmState struct {
	rec      model.VMRecord
	run      Run
	end      *model.RunEnd
	state    string
	detail   string
	reruns   []time.Time
	rerunAt  time.Time
	guest    string
	ip       string
	stopping bool
	client   Guest
	// value: closed once the session being dialed holds a client, or ends.
	up      chan struct{}
	cancel  context.CancelFunc
	in      Stream
	wmu     sync.Mutex
	keys    string
	lastErr string
	gen     int
	asked   map[string]chan guestMessage
	seq     int
	// value: a server's restarts, when its run began, its health check as
	// last run and whether one runs now; whether its ssh connection opened.
	restarts model.Restarts
	bootAt   time.Time
	check    model.CheckStatus
	checking bool
	ssh      bool
	// value: a server's last check runs; the restart it walks, nil when
	// none; its uptime record holds it down; it was marked since this
	// engine began.
	latency []model.LatencyPoint
	walk    *model.RestartWalk
	down    bool
	marked  bool
	// value: the last report of its agents, and whether one came.
	agents   []agents.Agent
	reported bool
	// value: SPEC B5.5: why its record holds it down, and the last lines
	// its console kept then; empty once it is back.
	cause     string
	lastLines []string
	// value: SPEC B5.10: when its create made it ready.
	readyAt time.Time
}

// invariant: SPEC B5.10: the create that made a VM tells its engine once it
// is ready: the engine keeps when, and tells the owner.
func (d *Engine) ready(ctx context.Context, vm string) error {
	v, err := d.find(vm)
	if err != nil {
		return err
	}
	d.mu.Lock()
	v.readyAt = d.h.Now().UTC()
	d.mu.Unlock()
	d.tell(ctx, vm, engine.Told{Kind: engine.NoticeVMReady, Parts: []string{vm + " is ready on " + d.m.ThisMacLabel()}})
	return nil
}

// invariant: at start, a live run is ADOPTED (never touched); a dead one or none
// leaves the workspace stopped and kept, never started again (SP-4): one its
// record wants running ended with its Mac. A server whose wanted state is
// Running starts with its Mac's engine: it is always on.
func (d *Engine) adopt(ctx context.Context) {
	records, _ := d.m.Records()
	for _, r := range records {
		v := &vmState{rec: r, state: engine.StateStopped, guest: engine.GuestAbsent}
		known, remembered := d.memory[r.Name]
		v.reruns, v.restarts, v.bootAt, v.check = known.Reruns, known.Restarts, known.BootAt, known.Check
		v.latency, v.walk, v.down = known.Latency, known.Restart, known.Down
		v.agents, v.reported = readReport(known.Agents), known.Reported
		v.cause, v.lastLines, v.readyAt = known.Cause, known.LastLines, known.ReadyAt
		d.mu.Lock()
		d.vms[r.Name] = v
		if len(known.LastHeard) > 0 {
			d.lastHeard[r.Name] = known.LastHeard
		}
		d.mu.Unlock()
		run, live := d.hv.Adopt(ctx, r.Name)
		switch {
		case live:
			v.run, v.state = run, engine.StateRunning
			d.connect(ctx, v)
			continue
		case remembered && (known.State == engine.StateBroken || known.State == engine.StateStale || !known.RerunAt.IsZero() || known.Detail == model.PoweredOffInside):
			v.state, v.detail, v.rerunAt = known.State, known.Detail, known.RerunAt
			continue
		case d.m.EngineRunning(ctx, r.Name):
			v.state = engine.StateRunning
			continue
		}
		switch model.Converge(r.Lifetime, r.Want == model.WantRunning, false) {
		case model.Boot:
			_ = d.start(ctx, v)
		case model.EndedWithItsMac:
			v.detail = endedWithItsMac
		case model.Adopt, model.Stay:
		}
	}
}

func (d *Engine) find(vm string) (*vmState, error) {
	d.mu.Lock()
	v, ok := d.vms[vm]
	d.mu.Unlock()
	if ok {
		return v, nil
	}
	records, _ := d.m.Records()
	for _, r := range records {
		if r.Name != vm {
			continue
		}
		v := &vmState{rec: r, state: engine.StateStopped, guest: engine.GuestAbsent}
		d.mu.Lock()
		if held, ok := d.vms[vm]; ok {
			v = held
		} else {
			d.vms[vm] = v
		}
		d.mu.Unlock()
		return v, nil
	}
	return nil, fault("unknown_vm", "unknown VM \""+vm+"\"", "station machines")
}

func (d *Engine) start(ctx context.Context, v *vmState) error {
	name := v.rec.Name
	if records, err := d.m.Records(); err == nil {
		for _, r := range records {
			if r.Name == name {
				d.mu.Lock()
				v.rec = r
				d.mu.Unlock()
			}
		}
	}
	d.refreshSelf(ctx)
	d.renderEnv(ctx, v)
	run, err := d.hv.Boot(ctx, name, model.Network{Bridged: v.rec.Network == model.Bridged})
	if err != nil {
		return err
	}
	d.mu.Lock()
	v.run, v.end, v.state, v.detail, v.stopping = run, nil, engine.StateRunning, "", false
	v.bootAt, v.check = d.h.Now(), model.CheckStatus{}
	if v.rec.Lifetime == model.AlwaysOn && v.down && v.walk == nil {
		v.walk = &model.RestartWalk{Step: 1, At: v.bootAt}
	}
	d.mu.Unlock()
	d.connect(ctx, v)
	return nil
}

func (d *Engine) tick(ctx context.Context) {
	records, _ := d.m.Records()
	known := map[string]bool{}
	for _, r := range records {
		known[r.Name] = true
		v, _ := d.find(r.Name)
		d.mu.Lock()
		v.rec = r
		ended := v.run != nil && !v.stopping && v.over()
		later := !v.rerunAt.IsZero() && !d.h.Now().Before(v.rerunAt)
		d.mu.Unlock()
		if ended {
			d.ended(ctx, v)
		}
		if later {
			d.mu.Lock()
			v.rerunAt = time.Time{}
			d.mu.Unlock()
			_ = d.start(ctx, v)
		}
		d.keep(ctx, v)
		d.walk(ctx, v)
		d.rehello(v)
	}
	d.mu.Lock()
	for name, v := range d.vms {
		if !known[name] && !v.stopping {
			if v.cancel != nil {
				v.cancel()
			}
			delete(d.vms, name)
		}
	}
	d.mu.Unlock()
	d.tickShares(ctx)
}

func (d *Engine) ended(ctx context.Context, v *vmState) {
	name := v.rec.Name
	d.mu.Lock()
	end := unread
	if v.end != nil {
		end = *v.end
	}
	now := d.h.Now()
	act, reruns := model.AfterRun(v.rec.Lifetime, end, v.reruns, now)
	v.reruns, v.run, v.end = reruns, nil, nil
	up := now.Sub(v.bootAt)
	d.mu.Unlock()
	d.disconnect(v)
	d.vmStopped(ctx, name)
	switch act {
	case model.ActRestart, model.ActRerun:
		d.wentDown(ctx, v, model.DownCause(end), end.LastLines, true, now)
	case model.ActPoweredOff, model.ActBroken, model.ActStale:
		d.wentDown(ctx, v, model.DownCause(end), end.LastLines, false, now)
	case model.ActDelete, model.ActStopped:
	}
	switch act {
	case model.ActRestart:
		d.restartLater(ctx, v, up, now)
	case model.ActPoweredOff:
		v.rec.Want = model.WantStopped
		_ = d.m.Save(v.rec)
		d.setState(v, engine.StateStopped, model.PoweredOffInside)
		d.tell(ctx, name, engine.PoweredOff(name))
	case model.ActRerun:
		if v.rec.Lifetime == model.AlwaysOn {
			d.mu.Lock()
			v.restarts = v.restarts.Rebooted(now)
			d.mu.Unlock()
		}
		_ = d.start(ctx, v)
	case model.ActDelete:
		d.releaseVM(ctx, name)
		d.forgetShares(ctx, name)
		_, _ = d.m.Remove(ctx, name)
		d.forget(name)
	case model.ActStopped:
		v.rec.Want = model.WantStopped
		_ = d.m.Save(v.rec)
		d.setState(v, engine.StateStopped, "")
	case model.ActBroken:
		d.setState(v, engine.StateBroken, model.BrokenDetail())
	case model.ActStale:
		d.setState(v, engine.StateStale, "the run ended with exit "+strconv.Itoa(end.ExitCode))
	}
}

func (d *Engine) setState(v *vmState, state, detail string) {
	d.mu.Lock()
	v.state, v.detail = state, detail
	d.mu.Unlock()
}

func (d *Engine) forget(name string) {
	d.mu.Lock()
	if v, ok := d.vms[name]; ok && v.cancel != nil {
		v.cancel()
	}
	delete(d.vms, name)
	delete(d.lastHeard, name)
	d.mu.Unlock()
}

func (d *Engine) vmViews() []engine.VMView {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]engine.VMView, 0, len(d.vms))
	for name, v := range d.vms {
		view := engine.VMView{
			Name: name, State: v.state, RunPID: v.runPID(), Holder: v.holder(), Guest: v.guest, IP: v.ip, Detail: v.detail, SSH: v.ssh, Restarts: v.restarts, Check: v.check,
			Latency: v.latency, Restart: v.walk, Cause: v.cause, LastLines: v.lastLines, ReadyAt: v.readyAt, Agent: agentView(v.agents),
		}
		if v.state == engine.StateRunning {
			view.BootAt = v.bootAt
		}
		out = append(out, view)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

const endedWithItsMac = "ended with its Mac"

func (d *Engine) opRun(ctx context.Context, vm string) error {
	v, err := d.find(vm)
	if err != nil {
		return err
	}
	if d.runs(v) {
		return nil
	}
	return d.start(ctx, v)
}

// invariant: run answers once the guest is connected (it has written the VM's
// keys block and services.env), or when the run ended first (its end is the
// supervision's), else guest_timeout after guestReady.
func (d *Engine) awaitGuest(ctx context.Context, vm string) error {
	v, err := d.find(vm)
	if err != nil {
		return err
	}
	for waited := time.Duration(0); waited < guestReady; waited += pollEvery {
		d.mu.Lock()
		connected, ended := v.guest == engine.GuestConnected, v.over()
		d.mu.Unlock()
		if connected || ended {
			return nil
		}
		if err := d.h.Sleep(ctx, pollEvery); err != nil {
			return err
		}
	}
	return fault("guest_timeout", vm+"'s guest did not connect in "+strconv.Itoa(int(guestReady.Seconds()))+" s", "station machines "+vm)
}

func (d *Engine) opStart(ctx context.Context, vm string) error {
	v, err := d.find(vm)
	if err != nil {
		return err
	}
	d.mu.Lock()
	v.reruns, v.rerunAt = nil, time.Time{}
	v.restarts.Recent, v.restarts.Streak, v.restarts.Looping = nil, 0, false
	d.mu.Unlock()
	if err := d.opRun(ctx, vm); err != nil {
		return err
	}
	v.rec.Want = model.WantRunning
	return d.m.Save(v.rec)
}

// invariant: a graceful stop is the guest's own poweroff, sent on the VM's one
// ssh connection (the NAT ssh when it is down); the hard stop comes only after
// the wait; the run's end is the daemon's own, so it is never read as a reboot.
func (d *Engine) opStop(ctx context.Context, vm string, want bool) error {
	v, err := d.find(vm)
	if err != nil {
		return err
	}
	return d.halt(ctx, v, want)
}

// invariant: halt ends a VM's run and keeps its disk; want also keeps the
// wish stopped, so a persistent VM stays stopped when the engine starts again.
func (d *Engine) halt(ctx context.Context, v *vmState, want bool) error {
	vm := v.rec.Name
	if want {
		v.rec.Want = model.WantStopped
		if err := d.m.Save(v.rec); err != nil {
			return err
		}
	}
	d.mu.Lock()
	v.stopping = true
	run, client := v.run, v.client
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		v.stopping = false
		d.mu.Unlock()
	}()
	if run == nil && !d.m.EngineRunning(ctx, vm) {
		d.disconnect(v)
		d.setState(v, engine.StateStopped, "")
		return nil
	}
	sent := false
	if client != nil {
		_, err := client.Exec(ctx, catalog.AsRoot("systemctl --no-block poweroff"), "")
		sent = err == nil
	}
	if !sent {
		if run == nil {
			if err := d.m.PowerOff(ctx, vm); err != nil {
				return err
			}
		} else {
			go func() { _ = d.m.PowerOff(context.WithoutCancel(ctx), vm) }()
		}
	}
	for waited := time.Duration(0); d.runs(v) && waited < stopWait; waited += pollEvery {
		if err := d.h.Sleep(ctx, pollEvery); err != nil {
			return err
		}
	}
	if d.runs(v) {
		_ = d.hv.Shutdown(ctx, vm, hardWait)
	}
	d.mu.Lock()
	v.run, v.end = nil, nil
	d.mu.Unlock()
	d.disconnect(v)
	d.vmStopped(ctx, vm)
	d.setState(v, engine.StateStopped, "")
	return nil
}

func (d *Engine) opDelete(ctx context.Context, vm string) error {
	v, err := d.find(vm)
	if err != nil {
		return err
	}
	d.mu.Lock()
	v.stopping = true
	d.mu.Unlock()
	d.vmStopped(ctx, vm)
	d.releaseVM(ctx, vm)
	d.forgetShares(ctx, vm)
	_, err = d.m.Remove(ctx, vm)
	d.forget(vm)
	return err
}

func (d *Engine) connect(ctx context.Context, v *vmState) {
	d.mu.Lock()
	if v.cancel != nil {
		d.mu.Unlock()
		return
	}
	if d.life != nil {
		ctx = d.life
	}
	gctx, cancel := context.WithCancel(ctx)
	v.gen++
	v.cancel, v.guest, v.up = cancel, engine.GuestConnecting, make(chan struct{})
	gen := v.gen
	d.mu.Unlock()
	go d.session(gctx, v, gen)
}

// invariant: a run that ended takes its guest session with it, so the next
// run's guest is awaited afresh: a stopped VM's ssh connection may outlive it,
// and a session it kept would answer for the new run before its sshd is up.
func (d *Engine) disconnect(v *vmState) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v.cancel != nil {
		v.cancel()
	}
	v.gen++
	v.cancel, v.guest, v.client, v.in, v.ssh = nil, engine.GuestAbsent, nil, nil, false
	v.release()
}

func (v *vmState) release() {
	if v.up != nil {
		close(v.up)
		v.up = nil
	}
}

func (d *Engine) current(v *vmState, gen int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return v.gen == gen
}

func (d *Engine) alive(v *vmState) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return v.run != nil && !v.stopping && !v.over()
}

func (d *Engine) session(ctx context.Context, v *vmState, gen int) {
	defer func() {
		d.mu.Lock()
		if v.gen == gen {
			v.cancel, v.guest, v.client, v.in = nil, engine.GuestAbsent, nil, nil
			v.release()
		}
		d.mu.Unlock()
	}()
	backoff := time.Second
	for d.alive(v) {
		d.guestOnce(ctx, v, gen)
		if !d.current(v, gen) || ctx.Err() != nil {
			return
		}
		d.vmStopped(ctx, v.rec.Name)
		if ctx.Err() != nil || d.h.Sleep(ctx, backoff) != nil {
			return
		}
		if backoff *= 2; backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (d *Engine) guestOnce(ctx context.Context, v *vmState, gen int) {
	name := v.rec.Name
	addr, err := d.hv.Address(ctx, name, 5*time.Second)
	if err != nil {
		d.guestFailed(ctx, v, "no ip: "+err.Error())
		return
	}
	ip := addr.String()
	d.mu.Lock()
	if v.gen == gen {
		v.ip = ip
	}
	d.mu.Unlock()
	at, err := netip.ParseAddrPort(joinPort(ip, d.o.GuestPort))
	if err != nil {
		d.guestFailed(ctx, v, err.Error())
		return
	}
	client, err := d.guests.Dial(ctx, name, at)
	if err != nil {
		d.guestFailed(ctx, v, err.Error())
		return
	}
	defer func() { _ = client.Close() }()
	d.mu.Lock()
	if v.gen != gen {
		d.mu.Unlock()
		return
	}
	session, err := client.Attach(ctx, engine.GuestSession)
	if err != nil {
		d.mu.Unlock()
		d.guestFailed(ctx, v, err.Error())
		return
	}
	v.client, v.ip, v.in, v.guest, v.ssh = client, ip, session, engine.GuestConnecting, true
	v.release()
	d.mu.Unlock()
	d.hello(v)
	scanner := bufio.NewScanner(session)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		d.fromGuest(ctx, v, scanner.Bytes())
	}
	_ = session.Close()
}

// invariant: why a guest does not connect reaches daemon.log once per new
// reason, so a guest stuck connecting is never a silent log.
func (d *Engine) guestFailed(ctx context.Context, v *vmState, reason string) {
	if ctx.Err() != nil {
		return
	}
	d.mu.Lock()
	repeated := v.lastErr == reason
	v.lastErr, v.guest, v.ssh = reason, engine.GuestAbsent, false
	d.mu.Unlock()
	if !repeated {
		d.logf(v.rec.Name+" guest", reason)
	}
}

func (d *Engine) write(v *vmState, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	v.wmu.Lock()
	defer v.wmu.Unlock()
	d.mu.Lock()
	in := v.in
	d.mu.Unlock()
	if in != nil {
		_, _ = in.Write(append(data, '\n'))
	}
}

// value: alias of hello v1, the line a guest of any version reads: its
// keys keep the words v2 wrote.
func (d *Engine) hello(v *vmState) {
	r := d.rightsOf(context.Background(), v.rec.Name)
	keys := d.hostKeys()
	d.mu.Lock()
	v.keys = strings.Join(keys, "\n")
	public := ports.ExposureOf(d.self.On, d.self.DNSName).PublicName
	d.mu.Unlock()
	env := v.rec.Env
	if env == nil {
		env = map[string]string{}
	}
	d.write(v, map[string]any{
		"v": 1, "type": "hello", "host": d.m.ThisMacName(), "vm": v.rec.Name, "public_host": public,
		"keys": keys, "env": env, "ignored": []int{d.o.GuestPort}, "min_port": r.Floor(),
	})
}

func (d *Engine) rehello(v *vmState) {
	d.mu.Lock()
	connected, was := v.guest == engine.GuestConnected, v.keys
	d.mu.Unlock()
	if connected && strings.Join(d.hostKeys(), "\n") != was {
		d.hello(v)
	}
}

type guestMessage struct {
	Type    string               `json:"type"`
	ID      string               `json:"id"`
	OK      bool                 `json:"ok"`
	Result  json.RawMessage      `json:"result"`
	Error   *enginecalls.OpError `json:"error"`
	Op      string               `json:"op"`
	Port    int                  `json:"port"`
	IP      string               `json:"ip"`
	Process string               `json:"process"`
	Quiet   bool                 `json:"quiet"`
	Ports   []struct {
		Port    int    `json:"port"`
		IP      string `json:"ip"`
		Process string `json:"process"`
		Quiet   bool   `json:"quiet"`
	} `json:"ports"`
	Agents []agents.Agent `json:"agents"`
}

func (d *Engine) fromGuest(ctx context.Context, v *vmState, line []byte) {
	var m guestMessage
	if ctx.Err() != nil || json.Unmarshal(line, &m) != nil {
		return
	}
	name := v.rec.Name
	switch m.Type {
	case "ready":
		d.takeBackBeforeReady(ctx, name)
		d.mu.Lock()
		v.guest, v.lastErr = engine.GuestConnected, ""
		d.mu.Unlock()
	case "snapshot":
		seen := map[int]bool{}
		heard := make([]ports.Listener, 0, len(m.Ports))
		quiet := map[int]ports.Listener{}
		for _, p := range m.Ports {
			l := ports.Listener{Port: p.Port, Bind: p.IP, Process: p.Process}
			if p.Quiet && !d.reserved(name, p.Port) {
				quiet[p.Port] = l
				continue
			}
			seen[p.Port] = true
			heard = append(heard, l)
			d.listen(ctx, name, p.Port, p.IP, p.Process)
		}
		d.setQuiet(name, quiet)
		d.hear(name, heard)
		d.unlistenExcept(name, seen)
	case "listen":
		l := ports.Listener{Port: m.Port, Bind: m.IP, Process: m.Process}
		if m.Quiet && !d.reserved(name, m.Port) {
			d.keepQuiet(name, l)
			return
		}
		d.hear(name, append(d.heardOf(name), l))
		d.listen(ctx, name, m.Port, m.IP, m.Process)
	case "unlisten":
		d.dropQuiet(name, m.Port)
		d.hear(name, slices.DeleteFunc(d.heardOf(name), func(l ports.Listener) bool { return l.Port == m.Port }))
		d.unlisten(name, m.Port)
	case "agents":
		d.heardAgents(ctx, v, m.Agents)
	case "reply":
		d.mu.Lock()
		ch := v.asked[m.ID]
		delete(v.asked, m.ID)
		d.mu.Unlock()
		if ch != nil {
			ch <- m
		}
	case "request":
		if m.Op == opAgentsHeld {
			go d.answerGuest(ctx, v, m)
			return
		}
		d.answerGuest(ctx, v, m)
	}
}

// value: the op whose answer reads every machine, its own guest included,
// so it is answered beside the guest's session, never in its way.
const opAgentsHeld = "agents-held"

func (d *Engine) answerGuest(ctx context.Context, v *vmState, m guestMessage) {
	result, err := d.guestRequest(ctx, v.rec.Name, m)
	reply := map[string]any{"v": 1, "type": "reply", "id": m.ID, "ok": err == nil}
	if err != nil {
		var op *enginecalls.OpError
		if !errors.As(err, &op) {
			op = fault("internal", err.Error(), "station ports")
		}
		reply["error"] = op
	} else {
		reply["result"] = result
	}
	d.write(v, reply)
}

func (d *Engine) guestRequest(ctx context.Context, vm string, m guestMessage) (any, error) {
	switch m.Op {
	case "ports":
		return d.surveyed(ctx, vm), nil
	case "reserve":
		// invariant: a guest's share is checked as the owner's is, on the
		// engine's side: its own binary is not trusted to have done it.
		if err := ports.Shareable(m.Port); err != nil {
			return nil, fault("ignored_port", err.Error(), "station never shares port 22 nor a port below 1024")
		}
		// invariant: a guest's share is judged by its VM's rights, never
		// made as the owner's: one they do not open is refused, one they
		// open takes their audience and is theirs to judge again.
		if r := d.rightsOf(ctx, vm); !r.Allows(m.Port) {
			return nil, fault("not_allowed", "port "+strconv.Itoa(m.Port)+" is outside the rights of "+vm+" ("+r.Said()+")", "on its Mac: station ports share "+vm+" "+strconv.Itoa(m.Port))
		}
		return d.shareOf(ctx, vm, m.Port, ports.ByRights)
	case "release":
		return d.stopSharing(ctx, vm, m.Port)
	case opAgentsHeld:
		return d.AgentsHeld(ctx)
	}
	return nil, fault("unknown_op", "unknown op \""+m.Op+"\"", "station ports")
}

func (d *Engine) askGuest(ctx context.Context, v *vmState, op string, budget time.Duration) (json.RawMessage, error) {
	ch := make(chan guestMessage, 1)
	d.mu.Lock()
	if v.asked == nil {
		v.asked = map[string]chan guestMessage{}
	}
	v.seq++
	id := "h" + strconv.Itoa(v.seq)
	v.asked[id] = ch
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(v.asked, id)
		d.mu.Unlock()
	}()
	d.write(v, map[string]any{"v": 1, "type": "request", "id": id, "op": op})
	wait, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	select {
	case m := <-ch:
		if !m.OK {
			if m.Error != nil {
				return nil, m.Error
			}
			return nil, fault("internal", v.rec.Name+"'s guest refused "+op, "station machines "+v.rec.Name)
		}
		return m.Result, nil
	case <-wait.Done():
		return nil, fault("guest_timeout", v.rec.Name+"'s guest did not answer "+op+" in "+budget.String(), "station machines "+v.rec.Name)
	}
}

func joinPort(ip string, port int) string {
	if strings.Contains(ip, ":") {
		return "[" + ip + "]:" + strconv.Itoa(port)
	}
	return ip + ":" + strconv.Itoa(port)
}

// invariant: a VM's environment is rendered at each of its runs, from its
// rights and Serve's state read then, never at an adoption, so it holds for
// the whole of a run, and a persistent VM picks up the public URLs at its
// first start after Serve turns on. A port an item sets an env for, inside
// the rights, is reserved then, before anything listens, so the server
// started on it reads its URL.
func (d *Engine) renderEnv(ctx context.Context, v *vmState) {
	r := d.loadRights(ctx, v.rec.Name)
	p := d.policy()
	d.mu.Lock()
	e := d.exposureLocked()
	env := map[string]string{}
	for _, h := range d.o.Ports {
		if len(h.Env) == 0 || !r.Allows(h.Port) {
			continue
		}
		l, i := ports.Find(d.reservations, v.rec.Name, h.Port)
		if i < 0 {
			reserved, err := d.reserveLocked(ctx, p, v.rec.Name, h.Port, r, ports.ByRights)
			if err != nil {
				continue
			}
			l = reserved
			d.reservations = append(d.reservations, l)
			d.saveReservations()
		}
		view := ports.ViewOf(l, e, p.HTTPS, ports.PhaseWaiting)
		for name, value := range ports.EnvOf(h.PortHint, view.URLs.Public, e.PublicName) {
			env[name] = value
		}
	}
	if len(env) > 0 {
		env[catalog.PublicNameEnv] = e.PublicName
	}
	rec := v.rec
	d.mu.Unlock()
	if maps.Equal(env, rec.Env) || len(env) == 0 && len(rec.Env) == 0 {
		return
	}
	rec.Env = env
	if d.m.Save(rec) == nil {
		d.mu.Lock()
		v.rec = rec
		d.mu.Unlock()
	}
}
