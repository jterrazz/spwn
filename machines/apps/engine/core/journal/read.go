package journal

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Query struct {
	// value: the day the read ends before, excluded; empty reads up to today.
	Before string
	// value: how many days before it the read covers.
	Days int
	// value: only the newest night group of the lines asked.
	Night bool
	// value: the read is narrowed to a Mac, a machine or a subject: its
	// groups carry no sum.
	Narrowed bool
	// value: the lines the read keeps; nil keeps every line.
	Keep func(Line) bool
}

func Kept(lines []Line, now time.Time) []Line {
	// invariant: a Mac keeps a line for Horizon: an older one is dropped, and
	// the rest are newest first, a tie in the order they were kept.
	from := now.Add(-Horizon)
	out := make([]Line, 0, len(lines))
	for _, l := range lines {
		if !l.At.Before(from) {
			out = append(out, l)
		}
	}
	slices.SortStableFunc(out, func(a, b Line) int { return b.At.Compare(a.At) })
	return out
}

func Between(lines []Line, since, before time.Time) []Line {
	// value: the lines at or after since and before before, a zero bound
	// none, newest first.
	out := make([]Line, 0, len(lines))
	for _, l := range lines {
		if (since.IsZero() || !l.At.Before(since)) && (before.IsZero() || l.At.Before(before)) {
			out = append(out, l)
		}
	}
	slices.SortStableFunc(out, func(a, b Line) int { return b.At.Compare(a.At) })
	return out
}

func DayOf(t, now time.Time) string { return t.In(now.Location()).Format(DayForm) }

func Words(day string, now time.Time) string {
	// value: a day in the engine's words: Today, Yesterday, else its weekday
	// and date ("Tuesday 11 November").
	today := now.In(now.Location())
	switch day {
	case today.Format(DayForm):
		return "Today"
	case today.AddDate(0, 0, -1).Format(DayForm):
		return "Yesterday"
	}
	t, err := time.ParseInLocation(DayForm, day, now.Location())
	if err != nil {
		return day
	}
	return t.Format("Monday 2 January")
}

type groupKey struct {
	day   string
	night bool
}

func keyOf(l Line, now time.Time) groupKey {
	if !l.Night.IsZero() {
		return groupKey{day: DayOf(l.Night, now), night: true}
	}
	return groupKey{day: DayOf(l.At, now)}
}

func Read(lines []Line, now time.Time, q Query) (groups []Group, next string) {
	// invariant: the lines of the horizon that Keep keeps, grouped by day,
	// a night's apart from its day's: the newest group first, a day's lines
	// newest first, a night's failed first, then done, then held. The lines
	// of one night that read alike (the same subject, upgrade and outcome)
	// are one line naming each machine; a failed one stays per machine, so
	// its retry names it. Night keeps only the newest night group; else the
	// read covers Days days before Before, and Next is the Before of the
	// next page: the day after the newest older day that holds a line.
	kept := make([]Line, 0, len(lines))
	for _, l := range Kept(lines, now) {
		if q.Keep == nil || q.Keep(l) {
			kept = append(kept, l)
		}
	}
	slices.SortStableFunc(kept, func(a, b Line) int {
		return cmp.Or(b.At.Compare(a.At), strings.Compare(a.Machine, b.Machine), strings.Compare(a.Text, b.Text))
	})
	newestNight := ""
	for _, l := range kept {
		if k := keyOf(l, now); k.night && k.day > newestNight {
			newestNight = k.day
		}
	}
	in := func(k groupKey) bool { return k.night && k.day == newestNight }
	if !q.Night {
		before := q.Before
		if before == "" {
			before = now.In(now.Location()).AddDate(0, 0, 1).Format(DayForm)
		}
		end, _ := time.ParseInLocation(DayForm, before, now.Location())
		from := end.AddDate(0, 0, -cmp.Or(q.Days, Days)).Format(DayForm)
		in = func(k groupKey) bool { return k.day >= from && k.day < before }
		older := ""
		for _, l := range kept {
			if k := keyOf(l, now); k.day < from && k.day > older {
				older = k.day
			}
		}
		if day, err := time.ParseInLocation(DayForm, older, now.Location()); err == nil {
			next = day.AddDate(0, 0, 1).Format(DayForm)
		}
	}
	var order []groupKey
	byKey := map[groupKey][]Line{}
	for _, l := range kept {
		k := keyOf(l, now)
		if !in(k) {
			continue
		}
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		byKey[k] = append(byKey[k], l)
	}
	groups = make([]Group, 0, len(order))
	for _, k := range order {
		g := Group{Day: k.day, Night: k.night, Title: Words(k.day, now)}
		if k.night {
			g.Title = "Night of " + g.Title
			if k.day == newestNight {
				g.Title = "Last night"
			}
			g.Lines = nightLines(byKey[k])
			if !q.Narrowed {
				g.Sum = sumOf(g.Lines)
			}
		} else {
			for _, l := range byKey[k] {
				g.Lines = append(g.Lines, readOf(l))
			}
		}
		groups = append(groups, g)
	}
	return groups, next
}

func readOf(l Line) ReadLine {
	out := ReadLine{At: l.At, Text: l.Text, By: l.By, Purpose: l.Purpose, On: []string{l.Machine}, Outcome: l.Outcome, Upgrade: l.Upgrade, Detail: l.Detail}
	if l.Subject != nil {
		s := *l.Subject
		out.Subject = &s
	}
	if l.Retry != nil {
		out.Retry = &Retry{Call: l.Retry.Call, Args: l.Retry.Args, On: l.Retry.On, Verb: "Retry"}
	}
	return out
}

func sameKind(l Line) string {
	subject := "text:" + l.Text
	if l.Subject != nil {
		subject = string(l.Subject.Kind) + ":" + l.Subject.Name
	}
	upgrade := ""
	if l.Upgrade != nil {
		upgrade = l.Upgrade.From + ">" + l.Upgrade.To
	}
	if l.Outcome == Failed {
		upgrade += "@" + l.Machine
	}
	return subject + "|" + upgrade + "|" + string(l.Outcome)
}

func nightLines(lines []Line) []ReadLine {
	// invariant: the group names who, so a night's line names no one.
	slices.SortStableFunc(lines, func(a, b Line) int {
		return cmp.Or(outcomeAt(a.Outcome)-outcomeAt(b.Outcome), a.At.Compare(b.At), strings.Compare(a.Text, b.Text), strings.Compare(a.Machine, b.Machine))
	})
	var out []ReadLine
	at := map[string]int{}
	for _, l := range lines {
		k := sameKind(l)
		if i, ok := at[k]; ok {
			if !slices.Contains(out[i].On, l.Machine) {
				out[i].On = append(out[i].On, l.Machine)
			}
			continue
		}
		r := readOf(l)
		r.By = ""
		at[k] = len(out)
		out = append(out, r)
	}
	return out
}

func outcomeAt(o Outcome) int {
	switch o {
	case Failed:
		return 0
	case Done:
		return 1
	case Held:
		return 2
	}
	return 3
}

func sumOf(lines []ReadLine) string {
	// value: what a night did in the engine's nouns: each line once,
	// whatever machines it names.
	moved, done, held, failed := 0, 0, 0, 0
	for _, l := range lines {
		switch {
		case l.Outcome == Failed:
			failed++
		case l.Outcome == Held:
			held++
		case l.Upgrade != nil:
			moved++
		default:
			done++
		}
	}
	var parts []string
	for _, c := range []struct {
		n    int
		noun string
	}{{moved, "moved"}, {done, "done"}, {held, "held"}, {failed, "failed"}} {
		if c.n > 0 {
			parts = append(parts, strconv.Itoa(c.n)+" "+c.noun)
		}
	}
	return strings.Join(parts, ", ")
}
