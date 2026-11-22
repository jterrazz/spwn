package awake

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

const (
	policyLine = "the always-on Mac policy"
	refusedBy  = "cannot be kept awake: "
)

func VerdictOf(r Report) *verdict.Verdict {
	// invariant: the cases of docs/reference/awake.md's verdict table, the
	// first that matches: none while the engine reads, the engine down,
	// then one per Mode, the Mac's own sleep timer before a guard that would
	// refuse a keep-awake. The sentence is Phrase's, the tone needs a look
	// wherever a page marks the Mac for one.
	if r.Engine == EngineReading {
		return nil
	}
	said := func(tone verdict.Tone, line string) *verdict.Verdict {
		p := r.Phrase()
		return &verdict.Verdict{Tone: tone, Sentence: strings.ToUpper(p[:1]) + p[1:] + ".", Line: line}
	}
	if r.EngineDown() {
		return said(verdict.Plain, "the engine is "+r.Engine)
	}
	switch r.Mode() {
	case NeverSleeps:
		if !r.NeverHeld {
			return said(verdict.Warn, policyLine)
		}
		return said(verdict.OK, policyLine)
	case AwakeUntil:
		if near := BatteryNear(r); near != "" {
			return said(verdict.Warn, near)
		}
		return said(verdict.OK, "")
	case StoppedByGuard:
		return said(verdict.Warn, "")
	case SleepsNormally:
		if r.MacNeverSleeps {
			return said(verdict.OK, "")
		}
		if v := Next(r.Caps, Held{}, r.Reading, 0); v.Tripped {
			return said(verdict.Warn, refusedBy+v.Trip.Words)
		}
	}
	return said(verdict.Plain, "")
}
