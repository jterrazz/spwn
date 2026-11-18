package machines

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

const LineNothingProvesDown = "Nothing proves it is down: it may be asleep, or off your network."

func UnseenVerdict(h MacView, now time.Time) *verdict.Verdict {
	// invariant: a Mac that does not answer is never said down, its tone its
	// standing's, plain asleep and warn otherwise: its silence says when it
	// was last seen, and sleep when its last answer was on battery with its
	// lid closed. A Mac that answers has none.
	s := MacStandingOf(h)
	switch s {
	case MacEngineStopped, MacEngineNotInstalled, MacOnline:
		return nil
	case MacAsleep, MacNotSeen:
	}
	v := verdict.Verdict{Tone: s.Tone(), Line: LineNothingProvesDown}
	switch {
	case h.Answered.IsZero():
		v.Sentence = h.Name + " has never answered."
	case s == MacAsleep:
		v.Sentence, v.Line = h.Name+" is asleep.", "It was on battery with its lid closed when last seen, "+Silence(now.Sub(h.Answered))+"."
	default:
		v.Sentence = h.Name + " has not been seen for " + Silence(now.Sub(h.Answered)) + "."
	}
	return &v
}

func Silence(d time.Duration) string {
	// value: a span as a page says it, a silence or how long a machine is
	// up: "1 min", "12 min", "3 h", "2 days".
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
	// answers its engine and how it sleeps.
	parts := []string{}
	switch s := MacStandingOf(h); s {
	case MacAsleep, MacNotSeen:
		return silentSentence(h, s, now)
	case MacEngineStopped:
		parts = append(parts, "Engine stopped.")
	case MacEngineNotInstalled:
		parts = append(parts, "Engine not installed.")
	case MacOnline:
		if h.Engine.Running() {
			parts = append(parts, "Engine running.")
		}
	}
	if h.Awake != "" {
		parts = append(parts, strings.ToUpper(h.Awake[:1])+h.Awake[1:]+".")
	}
	return strings.Join(parts, " ")
}

func silentSentence(h MacView, s MacStanding, now time.Time) string {
	switch {
	case h.Answered.IsZero():
		return "Never seen. Asleep or off the network."
	case s == MacAsleep:
		return "Asleep, not seen for " + Silence(now.Sub(h.Answered)) + "."
	}
	return "Not seen for " + Silence(now.Sub(h.Answered)) + ". Asleep or off the network."
}
