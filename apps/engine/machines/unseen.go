package machines

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

const LineNothingProvesDown = "Nothing proves it is down: it may be asleep, or off your network."

func UnseenVerdict(h MacView, now time.Time) *verdict.Verdict {
	// invariant: a Mac that does not answer is never said down, its tone
	// plain: its silence says when it was last seen, and sleep when its
	// last answer was on battery with its lid closed.
	if h.ThisMac || h.Reachable {
		return nil
	}
	v := verdict.Verdict{Tone: verdict.Plain, Line: LineNothingProvesDown}
	switch {
	case h.Answered.IsZero():
		v.Sentence = h.Name + " has never answered."
	case h.Asleep:
		v.Sentence, v.Line = h.Name+" is asleep.", "It was on battery with its lid closed when last seen, "+silence(now.Sub(h.Answered))+"."
	default:
		v.Sentence = h.Name + " has not been seen for " + silence(now.Sub(h.Answered)) + "."
	}
	return &v
}

func silence(d time.Duration) string {
	// value: a silence as a page says it: "1 min", "12 min", "3 h", "2 days".
	switch {
	case d < time.Hour:
		return strconv.Itoa(max(int(d/time.Minute), 1)) + " min"
	case d < 48*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + " h"
	}
	return strconv.Itoa(int(d/(24*time.Hour))) + " days"
}
