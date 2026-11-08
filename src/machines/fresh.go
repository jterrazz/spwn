package machines

import "time"

const StaleAfter = 30 * time.Second

const (
	ReachAnswers     = "answers"
	ReachAsleep      = "asleep"
	ReachUnreachable = "unreachable"
)

func Reach(h MacView) string {
	// value: SPEC B5.15: a Mac answers, sleeps (it last answered on battery
	// with its lid closed), or does not answer for a reason station cannot
	// tell, which needs a look.
	switch {
	case h.ThisMac || h.Reachable:
		return ReachAnswers
	case h.Asleep:
		return ReachAsleep
	}
	return ReachUnreachable
}

func Unheard(h MacView, now time.Time) bool {
	// value: SPEC B5.15: what is shown of a Mac is its last answer's, not
	// heard from for more than 30 s, or never heard from at all.
	if h.ThisMac || h.Reachable {
		return false
	}
	return h.Answered.IsZero() || now.Sub(h.Answered) > StaleAfter
}
