package calls

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const (
	pollEvery = 30 * time.Second
	keepEnd   = 24 * time.Hour
)

type Clock interface {
	// value: the time the keeper reads, handed by wire.
	Now() time.Time
}

// invariant: the first active guard past its threshold names the end; a
// guard is active only when this Mac reports the capability it reads.
func trip(c machine.Capabilities, r awake.Reading, s *awake.Session, heavyFor time.Duration) (reason, detail, words string) {
	held := awake.Held{}
	if s != nil {
		held = awake.Held{Lid: s.Lid, Battery: s.Battery}
	}
	v := awake.Next(c, held, r, heavyFor)
	if !v.Tripped {
		return "", "", ""
	}
	return v.Trip.Guard, v.Trip.Detail, v.Trip.Words
}

type KeeperOptions struct {
	Log func(message string)
	// value: a session a guard, its term or a stop ended; never power sleep.
	Notice func(awake.End)
	// value: what Never sleeps tells the owner: a guard it only notifies, or
	// the battery guard letting it go.
	Told   func(text string)
	Change func()
}

type Keeper struct {
	port  contract.Power
	clock Clock
	o     KeeperOptions

	mu      sync.Mutex
	session *awake.Session
	release func()
	// value: the name the session's assertion was taken under: the first
	// command's, kept while a later one refreshes the session.
	held       string
	lastEnd    *awake.End
	reading    awake.Reading
	heavySince time.Time
	polled     time.Time
	unreset    bool

	never        bool
	neverRelease func()
	neverSince   time.Time
	neverTrip    string
}

const neverSleepsHold = "station: never sleeps"

type absent struct{}

func (absent) Capabilities() contract.PowerCaps { return contract.PowerCaps{} }
func (absent) Read() (awake.Reading, error)     { return awake.Reading{}, contract.ErrUnsupported }
func (absent) Hold(string) (func(), error)      { return nil, contract.ErrUnsupported }
func (absent) Clamshell(bool) error             { return contract.ErrUnsupported }
func (absent) DisplaySleepNow(context.Context) error {
	return contract.ErrUnsupported
}
func (absent) Sleeping() bool { return false }
func (absent) Watch(context.Context) <-chan awake.Event {
	out := make(chan awake.Event)
	close(out)
	return out
}

func NewKeeper(port contract.Power, clock Clock, o KeeperOptions) *Keeper {
	if port == nil {
		port = absent{}
	}
	return &Keeper{port: port, clock: clock, o: o}
}

func (m *Keeper) caps() machine.Capabilities {
	return m.port.Capabilities().Onto(machine.Capabilities{})
}

func (m *Keeper) log(message string) {
	if m.o.Log != nil {
		m.o.Log(message)
	}
}

func (m *Keeper) changed() {
	if m.o.Change != nil {
		m.o.Change()
	}
}

func unsupported(err error) bool { return errors.Is(err, contract.ErrUnsupported) }

// invariant: clearing the clamshell mask is the safe direction, so a clear that
// fails is retried at every poll until it passes; a Mac with no power
// management has nothing to clear.
func (m *Keeper) clearClamshellLocked() {
	err := m.port.Clamshell(false)
	switch {
	case err == nil:
		m.unreset = false
	case unsupported(err):
		m.unreset = false
	default:
		m.unreset = true
		m.log("clearing the clamshell mask failed, retried in 30s: " + err.Error())
	}
}

func (m *Keeper) ClearClamshell() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearClamshellLocked()
}

func (m *Keeper) Held() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.session != nil
}

func (m *Keeper) View() awake.View {
	r, _ := m.port.Read()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reading = r
	v := awake.View{Reading: r, Guards: awake.GuardStates(r, m.session), NeverSleeps: m.never, NeverSince: m.neverSince, NeverHeld: m.neverRelease != nil, Sleeping: m.port.Sleeping()}
	if m.session != nil {
		s := *m.session
		v.Session = &s
	}
	if m.lastEnd != nil && m.clock.Now().Sub(m.lastEnd.At) < keepEnd {
		e := *m.lastEnd
		v.LastEnd = &e
	}
	return v
}

func flags(s awake.Session) string {
	var on []string
	if s.Lid {
		on = append(on, "lid")
	}
	if s.Battery {
		on = append(on, "battery")
	}
	if len(on) == 0 {
		return ""
	}
	return " (" + strings.Join(on, ", ") + ")"
}

func (m *Keeper) Awake(req awake.Request) (held awake.Session, refreshed bool, err error) {
	if !req.Until.IsZero() {
		req.For = req.Until.Sub(m.clock.Now().UTC().Truncate(time.Second))
		if req.For <= 0 {
			return awake.Session{}, false, &awake.RangeError{}
		}
	}
	if req.For == 0 {
		req.For = awake.DefaultFor
	}
	if bad := awake.CheckFor(req.For); bad != nil {
		return awake.Session{}, false, bad
	}
	r, err := m.port.Read()
	if unsupported(err) {
		return awake.Session{}, false, &awake.UnsupportedError{}
	}
	if err != nil {
		return awake.Session{}, false, err
	}
	want := awake.Session{Lid: req.Lid, Battery: req.Battery, By: req.By}
	if reason, _, words := trip(m.caps(), r, &want, 0); reason != "" {
		return awake.Session{}, false, &awake.GuardError{Guard: reason, Words: words}
	}
	m.mu.Lock()
	defer m.changed()
	defer m.mu.Unlock()
	m.reading = r
	now := m.clock.Now().UTC().Truncate(time.Second)
	want.StartedAt, want.Until, want.For = now, now.Add(req.For), req.For
	if !req.Until.IsZero() {
		want.Until = req.Until.UTC()
	}
	refreshed = m.session != nil
	if refreshed {
		was := m.session.Lid
		want.StartedAt = m.session.StartedAt
		if !req.Until.IsZero() && req.Until.Equal(m.session.Until) {
			// invariant: a switch flipped keeps the end the session holds.
			want.For = m.session.For
		}
		m.session = &want
		if want.Lid != was {
			if err := m.port.Clamshell(want.Lid); err != nil {
				m.log("clamshell: " + err.Error())
			}
		}
	} else {
		m.held = "station: " + req.By
		release, err := m.port.Hold(m.held)
		if err != nil {
			if unsupported(err) {
				return awake.Session{}, false, &awake.UnsupportedError{}
			}
			return awake.Session{}, false, err
		}
		m.release, m.session = release, &want
		m.heavySince, m.polled = time.Time{}, now
		if want.Lid {
			if err := m.port.Clamshell(true); err != nil {
				m.log("clamshell: " + err.Error())
			}
		}
	}
	m.log("awake until " + want.Until.Format(time.RFC3339) + flags(want) + " by " + want.By)
	return want, refreshed, nil
}

// invariant: every end releases the assertion and sets the clamshell back
// to normal, --lid or not, so whatever way a session ends the Mac sleeps as
// macOS decides.
func (m *Keeper) endLocked(reason, detail string) (awake.End, time.Duration) {
	now := m.clock.Now().UTC().Truncate(time.Second)
	after := now.Sub(m.session.StartedAt)
	if m.release != nil {
		m.release()
	}
	m.release, m.session, m.heavySince, m.held = nil, nil, time.Time{}, ""
	m.clearClamshellLocked()
	e := awake.End{At: now, Reason: reason, Detail: detail}
	m.lastEnd = &e
	m.log("ended (" + reason + ": " + detail + ")")
	return e, after
}

func (m *Keeper) finish(reason, detail string) bool {
	m.mu.Lock()
	if m.session == nil {
		m.mu.Unlock()
		return false
	}
	e, _ := m.endLocked(reason, detail)
	m.mu.Unlock()
	if reason != awake.ReasonSleep && m.o.Notice != nil {
		m.o.Notice(e)
	}
	m.changed()
	return true
}

func (m *Keeper) Sleep() awake.Slept {
	m.mu.Lock()
	if m.session == nil {
		m.mu.Unlock()
		return awake.Slept{}
	}
	_, after := m.endLocked(awake.ReasonSleep, "station awake off")
	m.mu.Unlock()
	m.changed()
	return awake.Slept{Ended: true, After: after}
}

func (m *Keeper) Stop() bool { return m.finish(awake.ReasonStop, "station engine stop") }

func (m *Keeper) Quit() {
	m.locked(m.releaseNeverLocked)
	if !m.finish(awake.ReasonQuit, "the engine stopped") {
		m.ClearClamshell()
	}
}

func term(d time.Duration) string {
	h, mins := int(d.Hours()), int(d.Minutes())%60
	switch {
	case h > 0 && mins > 0:
		return strconv.Itoa(h) + "h " + strconv.Itoa(mins) + "m"
	case h > 0:
		return strconv.Itoa(h) + "h"
	}
	return strconv.Itoa(mins) + "m"
}

func (m *Keeper) evaluateLocked(r awake.Reading) (reason, detail string) {
	now := m.clock.Now()
	m.reading = r
	if m.session == nil {
		return "", ""
	}
	if r.Thermal == awake.ThermalHeavy && r.LidClosedOnBattery() {
		if m.heavySince.IsZero() {
			m.heavySince = now
		}
	} else {
		m.heavySince = time.Time{}
	}
	heavyFor := time.Duration(0)
	if !m.heavySince.IsZero() {
		heavyFor = now.Sub(m.heavySince)
	}
	reason, detail, _ = trip(m.caps(), r, m.session, heavyFor)
	return reason, detail
}

type observed struct {
	policy, changed, lidHeld bool
	told, reason, detail     string
}

func (m *Keeper) observeLocked(ev awake.Event, r awake.Reading) observed {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil && m.never {
		told, changed := m.policyLocked(m.clock.Now())
		return observed{policy: true, told: told, changed: changed}
	}
	if reason, detail := m.evaluateLocked(r); reason != "" {
		return observed{reason: reason, detail: detail}
	}
	lidHeld := m.session != nil && m.session.Lid
	if lidHeld && (ev.Kind == "source" || ev.Kind == "lid") {
		// invariant: powerd may clear the mask (the safe direction), so it
		// is set again on every source and lid change while --lid holds.
		if err := m.port.Clamshell(true); err != nil {
			m.log("clamshell: " + err.Error())
		}
	}
	return observed{lidHeld: lidHeld}
}

func (m *Keeper) Observe(ctx context.Context, ev awake.Event) {
	r, err := m.port.Read()
	if err != nil {
		return
	}
	o := m.observeLocked(ev, r)
	switch {
	case o.policy:
		m.after(o.told, o.changed)
		return
	case o.reason != "":
		m.finish(o.reason, o.detail)
		return
	}
	if o.lidHeld && ev.Kind == "lid" && r.Lid == "closed" {
		m.sleepDisplay(ctx)
	}
	m.changed()
}

func (m *Keeper) sleepDisplay(ctx context.Context) {
	if err := m.port.DisplaySleepNow(ctx); err != nil {
		m.log("display: " + err.Error())
	}
}

func (m *Keeper) Tick() {
	// value: called every second by the daemon: the term is checked each
	// time; the guards, Low Power Mode's poll, the
	// heartbeat and a pending reset every 30 s.
	now := m.clock.Now()
	m.mu.Lock()
	if m.session != nil && !now.Before(m.session.Until) {
		detail := "its " + term(m.session.Until.Sub(m.session.StartedAt)) + " ended"
		m.mu.Unlock()
		m.finish(awake.ReasonUntil, detail)
		return
	}
	due := now.Sub(m.polled) >= pollEvery
	if !due {
		m.mu.Unlock()
		return
	}
	m.polled = now
	if m.unreset && m.session == nil {
		m.clearClamshellLocked()
	}
	if m.session == nil {
		told, changed := "", false
		if m.never {
			told, changed = m.policyLocked(now)
		}
		m.mu.Unlock()
		m.after(told, changed)
		return
	}
	m.mu.Unlock()
	r, err := m.port.Read()
	if err != nil {
		return
	}
	var reason, detail string
	var held, closed bool
	m.locked(func() {
		was := m.reading.Lid
		reason, detail = m.evaluateLocked(r)
		held = m.session != nil
		closed = held && m.session.Lid && was != "closed" && r.Lid == "closed"
	})
	if reason != "" {
		m.finish(reason, detail)
		return
	}
	if closed {
		// invariant: a lid closed between two polls sleeps the display as
		// a lid event does: a request, never a display assertion.
		m.sleepDisplay(context.Background())
	}
	if held {
		m.log("held " + heartbeat(r))
		m.changed()
	}
}

func heartbeat(r awake.Reading) string {
	lid := r.Lid
	if lid == "" {
		lid = "none"
	}
	battery := "none"
	if r.HasBattery {
		battery = strconv.Itoa(r.Percent) + "%"
	}
	return "lid=" + lid + " source=" + r.Source + " battery=" + battery + " thermal=" + strconv.Itoa(r.Thermal)
}

func (m *Keeper) Policy(never bool, since time.Time) {
	// invariant: Never sleeps is the engine's one assertion of the policy: held
	// while the preference says so, except while the battery guard of a Mac
	// that reports a battery trips; every other guard only tells the owner, once
	// per trip.
	var told string
	var changed bool
	m.locked(func() {
		m.never = never
		if since.IsZero() {
			since = m.clock.Now()
		}
		told, changed = m.policyLocked(since)
	})
	m.after(told, changed)
}

// invariant: a stretch that runs code under m.mu holds it through locked,
// whose unlock is deferred, so a panic a boundary recovers leaves it free.
func (m *Keeper) locked(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f()
}

func (m *Keeper) after(told string, changed bool) {
	if told != "" && m.o.Told != nil {
		m.o.Told(told)
	}
	if changed {
		m.changed()
	}
}

func (m *Keeper) policyLocked(since time.Time) (told string, changed bool) {
	if !m.never {
		if m.neverRelease == nil && m.neverSince.IsZero() {
			return "", false
		}
		m.releaseNeverLocked()
		m.neverSince, m.neverTrip = time.Time{}, ""
		m.log("never sleeps: off")
		return "", true
	}
	r, err := m.port.Read()
	if err != nil {
		return "", false
	}
	m.reading = r
	v := awake.Next(m.caps(), awake.Held{NeverSleeps: true}, r, 0)
	trip := ""
	if v.Tripped {
		trip = v.Trip.Guard
	}
	if trip != m.neverTrip {
		m.neverTrip = trip
		switch {
		case v.Tripped && v.Ends:
			told = "Never sleeps let go: " + v.Trip.Detail
		case v.Tripped:
			told = "Never sleeps stays on: " + v.Trip.Detail
		}
		changed = true
	}
	if v.Tripped && v.Ends {
		if m.neverRelease != nil {
			m.releaseNeverLocked()
			m.log("never sleeps: let go (" + v.Trip.Detail + ")")
			changed = true
		}
		return told, changed
	}
	if m.neverRelease == nil {
		release, err := m.port.Hold(neverSleepsHold)
		if err != nil {
			if !unsupported(err) {
				m.log("never sleeps: " + err.Error())
			}
			return told, changed
		}
		m.neverRelease = release
		if m.neverSince.IsZero() {
			m.neverSince = since.UTC().Truncate(time.Second)
		}
		m.log("never sleeps: held")
		changed = true
	}
	return told, changed
}

func (m *Keeper) releaseNeverLocked() {
	if m.neverRelease != nil {
		m.neverRelease()
		m.neverRelease = nil
	}
}

func (m *Keeper) NeverSleeps() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.never
}

func (m *Keeper) Kept() awake.Kept {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := awake.Kept{NeverSince: m.neverSince, NeverTrip: m.neverTrip, HeavySince: m.heavySince, Lid: m.reading.Lid, NeverHeld: m.neverRelease != nil, Held: m.held}
	if m.session != nil {
		s := *m.session
		k.Session = &s
	}
	if m.lastEnd != nil {
		e := *m.lastEnd
		k.LastEnd = &e
	}
	return k
}

func (m *Keeper) Restore(k awake.Kept) {
	// invariant: Restore follows Begin's reset: a kept keep-awake is held again
	// with its lid as it was, so a stepped engine is the one long engine it
	// stands for.
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastEnd, m.neverSince, m.neverTrip, m.heavySince = k.LastEnd, k.NeverSince, k.NeverTrip, k.HeavySince
	m.reading.Lid = k.Lid
	if k.NeverHeld {
		if release, err := m.port.Hold(neverSleepsHold); err == nil {
			m.neverRelease = release
		}
	}
	if k.Session == nil {
		return
	}
	held := k.Held
	if held == "" {
		held = "station: " + k.Session.By
	}
	release, err := m.port.Hold(held)
	if err != nil {
		return
	}
	m.held = held
	s := *k.Session
	m.release, m.session, m.polled = release, &s, time.Time{}
	if s.Lid {
		if err := m.port.Clamshell(true); err != nil {
			m.log("clamshell: " + err.Error())
		}
	}
}

func (m *Keeper) Suspend() {
	// invariant: Suspend is a stepped engine's pause between two commands: its
	// holds stay as the long engine keeps them, only forgotten by this one.
	m.mu.Lock()
	defer m.mu.Unlock()
	m.release, m.neverRelease, m.session = nil, nil, nil
}
