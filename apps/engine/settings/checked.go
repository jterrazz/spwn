package settings

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type CheckedState string

const (
	AsExpected               CheckedState = "as-expected"
	NeedsFix                 CheckedState = "needs-fix"
	CheckedOffThisMac        CheckedState = "off-here"
	CheckedNotForThisMachine CheckedState = "not-for-this-machine"
	CheckedUnknown           CheckedState = "unknown"
)

const (
	lockWithin   = 5 * time.Minute
	backupWithin = 7 * 24 * time.Hour
)

func (r Row) CheckedItem() catalog.CheckedSetting {
	for _, it := range catalog.CheckedSettings {
		if it.Name == r.Name {
			return it
		}
	}
	return catalog.CheckedSetting{Name: r.Name}
}

func CheckedNamed(name string) (catalog.CheckedSetting, bool) {
	for _, it := range catalog.CheckedSettings {
		if it.Name == name {
			return it, true
		}
	}
	return catalog.CheckedSetting{}, false
}

type Expectation struct {
	// value: what this machine should have, from what it reports and the
	// owner's policy, never from its model name. Laptop: it reports a battery.
	Laptop bool
	// value: Never sleeps is applied on this Mac.
	AlwaysOn bool
	Now      time.Time
}

func Expect(caps machine.Capabilities, alwaysOn bool, now time.Time) Expectation {
	return Expectation{Laptop: caps.Battery, AlwaysOn: alwaysOn, Now: now}
}

func JudgeChecked(it catalog.CheckedSetting, lives bool, facts map[catalog.Fact]Fact, expect Expectation, off bool) Row {
	// invariant: one checked setting on one machine, from its place, the
	// owner's Off here and the facts macOS answered: a fact it could not read
	// is Unknown, never Needs a fix; a fix is where the owner changes it,
	// never a station command.
	row := Row{Name: it.Name, Group: Checked}
	switch {
	case !lives:
		row.Checked, row.Why = CheckedNotForThisMachine, LivesOnReason(it.Kinds)
		return row
	case off:
		row.Checked = CheckedOffThisMac
		return row
	}
	for _, f := range it.Facts {
		if got, ok := facts[f]; !ok || got.State == catalog.StateUnknown || got.State == catalog.StatePending {
			if f != catalog.FactDisplaySleep || expect.Laptop {
				row.Checked, row.Value = CheckedUnknown, unknownWhy(got)
				return row
			}
		}
	}
	fact := facts[it.Facts[0]]
	var v verdict
	switch it.Facts[0] {
	case catalog.FactScreenLock:
		v = screenLock(fact, facts[catalog.FactDisplaySleep], expect.Laptop)
	case catalog.FactRemoteLogin:
		v = remoteLogin(fact.Value == "on", expect)
	case catalog.FactTimeMachine:
		v = lastBackup(fact, expect.Now)
	case catalog.FactSystemUpdates:
		v = verdict{ok: fact.State == catalog.StateOK, value: waiting(fact)}
	default:
		v = verdict{ok: fact.Value == "on", value: fact.Value}
	}
	row.Checked, row.Value, row.Expected = AsExpected, v.value, v.expected
	if !v.ok {
		row.Checked, row.Fix = NeedsFix, finding.Fix{Path: it.Fix}
	}
	return row
}

type verdict struct {
	ok              bool
	value, expected string
}

func unknownWhy(f Fact) string {
	if f.Detail != "" {
		return f.Detail
	}
	return "could not be read"
}

// invariant: a laptop locks within 5 min of idle: the display's idle time
// and the lock's delay after it; a desktop's display may stay on, so it is
// judged on the delay after the display turns off alone.
func screenLock(lock, display Fact, laptop bool) verdict {
	const expected = "expected 5 min"
	if lock.Value == "off" {
		return verdict{value: "never locks", expected: expected}
	}
	after := time.Duration(seconds(lock.Value)) * time.Second
	words := "locks " + minutes(after) + " after the display turns off"
	switch idle := seconds(display.Value); {
	case !laptop && after == 0:
		return verdict{ok: true, value: "locks when the display turns off"}
	case laptop && idle == 0:
		return verdict{value: "never locks: the display never turns off", expected: expected}
	case laptop:
		after += time.Duration(idle) * time.Minute
		words = "locks after " + minutes(after)
	}
	if after > lockWithin {
		return verdict{value: words, expected: expected}
	}
	return verdict{ok: true, value: words}
}

func seconds(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return max(n, 0)
}

func minutes(d time.Duration) string {
	if d < time.Minute {
		return strconv.Itoa(int(d.Seconds())) + " s"
	}
	return strconv.Itoa(int(d.Round(time.Minute).Minutes())) + " min"
}

// invariant: Remote login is on on an always-on Mac (it is reached from
// afar), off on a laptop that is not one; any other Mac has no expectation.
func remoteLogin(on bool, expect Expectation) verdict {
	v := verdict{ok: true, value: "off"}
	if on {
		v.value = "on"
	}
	switch {
	case expect.AlwaysOn && !on:
		v.ok, v.expected = false, "expected on an always-on Mac"
	case !expect.AlwaysOn && expect.Laptop && on:
		v.ok, v.expected = false, "not expected on a laptop"
	}
	return v
}

func lastBackup(f Fact, now time.Time) verdict {
	at, err := time.Parse(time.RFC3339, f.Value)
	if err != nil {
		return verdict{value: "no backup yet"}
	}
	age := now.Sub(at)
	return verdict{ok: age <= backupWithin, value: "last backup " + ago(age)}
}

func ago(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return strconv.Itoa(int(d/(24*time.Hour))) + " d ago"
	case d >= time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + " h ago"
	}
	return strconv.Itoa(max(int(d/time.Minute), 0)) + " min ago"
}

func waiting(f Fact) string {
	if f.State == catalog.StateOK {
		return "up to date"
	}
	n, _, _ := strings.Cut(f.Value, " ")
	if n == "1" {
		return "1 update waiting"
	}
	return n + " updates waiting"
}
