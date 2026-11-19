package machines

import (
	"testing"
	"time"
)

func at(day, hour, minute int) time.Time {
	return time.Date(2026, 11, day, hour, minute, 0, 0, time.Local)
}

func TestADowntimeAcrossMidnightIsSplitBetweenItsDays(t *testing.T) {
	u, _ := Uptime{}.Marked(UptimeMark{Change: WentDown, At: at(5, 23, 50), Cause: "out of memory"})
	u, _ = u.Marked(UptimeMark{Change: CameBack, At: at(6, 0, 5), Restarted: true})
	want := []UptimeDay{{Date: "2026-11-05", DownS: 600}, {Date: "2026-11-06", DownS: 300, Restarts: 1}}
	if len(u.Days) != 2 || u.Days[0] != want[0] || u.Days[1] != want[1] {
		t.Errorf("days = %+v; want %+v", u.Days, want)
	}
	view, _ := u.View(at(6, 12, 0))
	if got := view.Cells[len(view.Cells)-2:]; got[0].Mark != DayDown || got[1].Mark != DayRestart {
		t.Errorf("the last two days are %s and %s; want down (held at its midnight) then restart", got[0].Mark, got[1].Mark)
	}
}

func TestAMarkThatChangesNothingSaysSo(t *testing.T) {
	u, changed := Uptime{}.Marked(UptimeMark{Change: CameBack, At: at(5, 9, 0)})
	if !changed || !u.Since.Equal(at(5, 9, 0)) {
		t.Fatalf("the first mark = %+v, %v; want the record started", u, changed)
	}
	if _, changed = u.Marked(UptimeMark{Change: CameBack, At: at(5, 10, 0)}); changed {
		t.Errorf("back while up changed the record")
	}
	down, _ := u.Marked(UptimeMark{Change: WentDown, At: at(5, 11, 0)})
	if again, changed := down.Marked(UptimeMark{Change: WentDown, At: at(5, 11, 5)}); changed || len(again.Incidents) != 1 {
		t.Errorf("down while down = %+v, %v; want the first incident alone", again.Incidents, changed)
	}
}

func TestTheRecordKeepsThirtyDaysAndTwentyIncidents(t *testing.T) {
	u := Uptime{}
	for day := range 40 {
		down := at(1, 9, 0).AddDate(0, 0, day)
		u, _ = u.Marked(UptimeMark{Change: WentDown, At: down})
		u, _ = u.Marked(UptimeMark{Change: CameBack, At: down.Add(time.Minute)})
	}
	if len(u.Days) != UptimeDays || u.Days[0].Date != "2026-11-11" || len(u.Incidents) != UptimeIncidents {
		t.Errorf("kept %d days from %s and %d incidents; want 30 from 2026-11-11 and 20", len(u.Days), u.Days[0].Date, len(u.Incidents))
	}
}

func TestTheThirtyDaysAreSaidDayByDayAndInOneSentence(t *testing.T) {
	u, _ := Uptime{}.Marked(UptimeMark{Change: CameBack, At: at(3, 9, 0)})
	u, _ = u.Marked(UptimeMark{Change: WentDown, At: at(10, 14, 5), Cause: "out of memory"})
	u, _ = u.Marked(UptimeMark{Change: CameBack, At: at(10, 14, 9), Restarted: true})
	u, _ = u.Marked(UptimeMark{Change: WentDown, At: at(12, 8, 0), Cause: CheckFailed})
	strip, ok := u.Strip(at(12, 10, 0))
	if !ok || len(strip.Days) != UptimeDays {
		t.Fatalf("strip = %+v, %v; want %d days", strip, ok, UptimeDays)
	}
	wantSentence := "Up 99.04 % over 10 days, 1 restart"
	if strip.Sentence != wantSentence {
		t.Errorf("sentence = %q; want %q", strip.Sentence, wantSentence)
	}
	last := len(strip.Days) - 1
	for _, c := range []struct {
		at   int
		want string
	}{
		{0, "Wed 14 Oct. Not recorded yet"},
		{last - 9, "Tue 3 Nov. Up all day"},
		{last - 2, "Tue 10 Nov. Down from 14:05 to 14:09, out of memory; restarted 1 time"},
		{last, "Thu 12 Nov. Down from 08:00 to now, its check failed"},
	} {
		if got := strip.Days[c.at].Sentence; got != c.want {
			t.Errorf("day %d = %q; want %q", c.at, got, c.want)
		}
	}
}
