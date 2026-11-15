package machines

import (
	"strconv"
	"strings"
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

func SentenceOf(h MacView, now time.Time) string {
	// value: the one sentence a Mac is told by in the list of machines: a
	// Mac that does not answer says since when it is not seen, one that
	// answers its engine and what keeps it awake.
	if h.ThisMac || h.Reachable {
		parts := []string{}
		switch h.Engine.State {
		case EngineRunning:
			parts = append(parts, "Engine running.")
		case EngineStopped:
			parts = append(parts, "Engine stopped.")
		case EngineNotInstalled:
			parts = append(parts, "Engine not installed.")
		}
		if h.Awake != "" {
			parts = append(parts, strings.ToUpper(h.Awake[:1])+h.Awake[1:]+".")
		}
		return strings.Join(parts, " ")
	}
	switch {
	case h.Answered.IsZero():
		return "Never seen. Asleep or off the network."
	case h.Asleep:
		return "Asleep, not seen for " + silence(now.Sub(h.Answered)) + "."
	}
	return "Not seen for " + silence(now.Sub(h.Answered)) + ". Asleep or off the network."
}
