package machines

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// value: SPEC B5.2: the days a server's record keeps, and its incidents.
	UptimeDays      = 30
	UptimeIncidents = 20
	dateLayout      = "2006-01-02"
)

type UptimeChange string

const (
	// value: what the keep job marks on a server's record: its run ended
	// or its check failed, or it is healthy again.
	WentDown UptimeChange = "down"
	CameBack UptimeChange = "back"
)

type UptimeMark struct {
	Server string
	Change UptimeChange
	At     time.Time
	// value: why it went down, in the words its end says.
	Cause string
	// value: it came back through a restart, counted on the day it did.
	Restarted bool
}

const (
	// value: why a server went down when its check failed and its
	// console told nothing more.
	CheckFailed = "its check failed"
)

type UptimeDay struct {
	Date     string `json:"date"`
	DownS    int64  `json:"down_s,omitempty"`
	Restarts int    `json:"restarts,omitempty"`
}

type Incident struct {
	From time.Time `json:"from"`
	// value: zero while the server is still down.
	To    time.Time `json:"to,omitzero"`
	Cause string    `json:"cause,omitempty"`
}

type Uptime struct {
	// value: when the record started; nothing before it is back-filled.
	Since     time.Time   `json:"since,omitzero"`
	Days      []UptimeDay `json:"days,omitempty"`
	Incidents []Incident  `json:"incidents,omitempty"`
}

func (u Uptime) Marked(m UptimeMark) (Uptime, bool) {
	// invariant: SPEC B5.2: the first mark starts the record; going down
	// opens one incident, coming back closes it and adds its down seconds to
	// each day it covered, a restart counted on the day it came back. A mark
	// that changes nothing (down while down, back while up) says so, and
	// only the last 30 days and 20 incidents are kept.
	out := Uptime{Since: u.Since, Days: slices.Clone(u.Days), Incidents: slices.Clone(u.Incidents)}
	changed := false
	if out.Since.IsZero() {
		out.Since, changed = m.At, true
	}
	open := out.open()
	switch m.Change {
	case WentDown:
		if open < 0 {
			out.Incidents = append(out.Incidents, Incident{From: m.At, Cause: m.Cause})
			changed = true
		}
	case CameBack:
		if open >= 0 {
			out.Incidents[open].To = m.At
			out.addDown(out.Incidents[open].From, m.At)
			if m.Restarted {
				out.day(dateOf(m.At)).Restarts++
			}
			changed = true
		}
	}
	first := dateOf(firstCell(m.At))
	out.Days = slices.DeleteFunc(out.Days, func(d UptimeDay) bool { return d.Date < first })
	if n := len(out.Incidents); n > UptimeIncidents {
		out.Incidents = out.Incidents[n-UptimeIncidents:]
	}
	return out, changed
}

func (u Uptime) open() int {
	for i, in := range u.Incidents {
		if in.To.IsZero() {
			return i
		}
	}
	return -1
}

// invariant: the down seconds of an incident are split at each local
// midnight it crossed.
func (u *Uptime) addDown(from, to time.Time) {
	for t := from; t.Before(to); {
		end := earlier(midnight(t).AddDate(0, 0, 1), to)
		u.day(dateOf(t)).DownS += int64(end.Sub(t) / time.Second)
		t = end
	}
}

func (u *Uptime) day(date string) *UptimeDay {
	i := slices.IndexFunc(u.Days, func(d UptimeDay) bool { return d.Date == date })
	if i < 0 {
		u.Days = append(u.Days, UptimeDay{Date: date})
		slices.SortFunc(u.Days, func(a, b UptimeDay) int { return strings.Compare(a.Date, b.Date) })
		i = slices.IndexFunc(u.Days, func(d UptimeDay) bool { return d.Date == date })
	}
	return &u.Days[i]
}

type DayMark string

const (
	DayOK      DayMark = "ok"
	DayRestart DayMark = "restart"
	DayDown    DayMark = "down"
	DayNone    DayMark = "none"
)

type UptimeCell struct {
	Date string
	Mark DayMark
	// value: what its tooltip tells: each incident the day held, and the
	// restarts it counted.
	Down     []Incident `json:",omitempty"`
	Restarts int        `json:",omitempty"`
	// value: the day said in one sentence, its date first; judged by
	// machines.list, empty in the view.
	Sentence string `json:",omitempty"`
}

type UptimeStrip struct {
	// value: the 30 days said in one sentence, how long it was up and the
	// restarts it counted: `Up 99.93 % over 30 days, 2 restarts`.
	Sentence     string
	Availability float64
	Days         []UptimeCell
}

func (c UptimeCell) Tip(clock func(time.Time) string) string {
	// invariant: a day says each incident it held, from when to when and
	// why, then its restarts; a day with none is up all day, and a day
	// before the record says nothing.
	var parts []string
	for _, in := range c.Down {
		words := "Down from " + clock(in.From) + " to "
		if in.To.IsZero() {
			words += "now"
		} else {
			words += clock(in.To)
		}
		if in.Cause != "" {
			words += ", " + in.Cause
		}
		parts = append(parts, words)
	}
	if c.Restarts > 0 {
		parts = append(parts, "restarted "+counted(c.Restarts, "time"))
	}
	if len(parts) == 0 && c.Mark == DayOK {
		return "Up all day"
	}
	return strings.Join(parts, "; ")
}

func counted(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

func (u Uptime) Strip(now time.Time) (UptimeStrip, bool) {
	// invariant: each day of the view said by Tip, behind its date as the
	// Mac's calendar writes it, a day before the record as not recorded;
	// the sentence counts the days the record holds and the restarts of
	// them.
	view, ok := u.View(now)
	if !ok {
		return UptimeStrip{}, false
	}
	clock := func(t time.Time) string { return t.In(now.Location()).Format("15:04") }
	strip := UptimeStrip{Availability: view.Availability, Days: view.Cells}
	recorded, restarts := 0, 0
	for i, c := range view.Cells {
		tip := c.Tip(clock)
		if c.Mark == DayNone {
			tip = "Not recorded yet"
		} else {
			recorded++
			restarts += c.Restarts
		}
		day, _ := time.ParseInLocation(dateLayout, c.Date, time.Local)
		strip.Days[i].Sentence = day.Format("Mon 2 Jan") + ". " + tip
	}
	strip.Sentence = "Up " + strconv.FormatFloat(view.Availability, 'f', -1, 64) + " % over " + counted(recorded, "day")
	if restarts > 0 {
		strip.Sentence += ", " + counted(restarts, "restart")
	}
	return strip, true
}

type UptimeView struct {
	Since time.Time
	// value: the last 30 days, the oldest first, today last.
	Cells []UptimeCell
	// value: the percent of the time recorded in those days it was up,
	// two decimals, rounded down so a down minute never reads 100.
	Availability float64
}

func (u Uptime) View(now time.Time) (UptimeView, bool) {
	// invariant: SPEC B5.2: a day before the record is none; a day an
	// incident still held at its end (or holds now, today) is down; a day
	// whose downtime all recovered is restart, so today turns restart the
	// moment a down server is back; any other day is ok.
	if u.Since.IsZero() {
		return UptimeView{}, false
	}
	v := UptimeView{Since: u.Since, Cells: make([]UptimeCell, 0, UptimeDays)}
	since := dateOf(u.Since)
	for i := range UptimeDays {
		start := firstCell(now).AddDate(0, 0, i)
		date := dateOf(start)
		if date < since {
			v.Cells = append(v.Cells, UptimeCell{Date: date, Mark: DayNone})
			continue
		}
		v.Cells = append(v.Cells, u.cell(date, start, start.AddDate(0, 0, 1), now))
	}
	v.Availability = u.availability(now)
	return v, true
}

func (u Uptime) cell(date string, start, end, now time.Time) UptimeCell {
	c := UptimeCell{Date: date, Mark: DayOK}
	held := false
	for _, in := range u.Incidents {
		if !in.From.Before(earlier(end, now)) || (!in.To.IsZero() && !in.To.After(start)) {
			continue
		}
		c.Down = append(c.Down, in)
		held = held || in.To.IsZero() || !in.To.Before(end)
	}
	i := slices.IndexFunc(u.Days, func(d UptimeDay) bool { return d.Date == date })
	if i >= 0 {
		c.Restarts = u.Days[i].Restarts
	}
	switch {
	case held:
		c.Mark = DayDown
	case len(c.Down) > 0 || (i >= 0 && (u.Days[i].DownS > 0 || u.Days[i].Restarts > 0)):
		c.Mark = DayRestart
	}
	return c
}

func (u Uptime) availability(now time.Time) float64 {
	start := later(firstCell(now), u.Since)
	recorded := now.Sub(start)
	if recorded <= 0 {
		return 100
	}
	first := dateOf(firstCell(now))
	var down time.Duration
	for _, d := range u.Days {
		if d.Date >= first {
			down += time.Duration(d.DownS) * time.Second
		}
	}
	if i := u.open(); i >= 0 {
		down += now.Sub(later(u.Incidents[i].From, start))
	}
	up := 100 * (1 - float64(down)/float64(recorded))
	return math.Max(0, math.Floor(up*100)/100)
}

func earlier(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func firstCell(now time.Time) time.Time { return midnight(now).AddDate(0, 0, 1-UptimeDays) }

func midnight(t time.Time) time.Time {
	t = t.Local()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func dateOf(t time.Time) string { return t.Local().Format(dateLayout) }
