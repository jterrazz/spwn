package ports

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

const (
	SentenceNothingListens = "Nothing listens on this machine."
	LineNothingListens     = "A port is listed as soon as something listens on it."
	LineExposed            = "Declare it public, or turn it off in the app that opens it."
)

func VerdictOf(f AllMachines) verdict.Verdict {
	// invariant: the Ports page's head: a port exposed first, then what
	// listens with what is shared as its line; an empty listing says so.
	listening, shared, exposed := 0, 0, 0
	first := 0
	for _, h := range f.Macs {
		listening += len(h.Ports)
		shared += len(h.Shared)
		for _, p := range h.Ports {
			if p.Red() {
				if exposed == 0 {
					first = p.Port
				}
				exposed++
			}
		}
	}
	switch {
	case exposed == 1:
		return verdict.Verdict{Tone: verdict.Bad, Sentence: "Port " + strconv.Itoa(first) + " is exposed.", Line: LineExposed}
	case exposed > 1:
		return verdict.Verdict{Tone: verdict.Bad, Sentence: strconv.Itoa(exposed) + " ports are exposed.", Line: LineExposed}
	case listening == 0 && shared == 0:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: SentenceNothingListens, Line: LineNothingListens}
	}
	sentence := strconv.Itoa(listening) + " ports listen."
	if listening == 1 {
		sentence = "One port listens."
	}
	line := strconv.Itoa(shared) + " are shared."
	switch shared {
	case 0:
		line = "None is shared."
	case 1:
		line = "One is shared."
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: sentence, Line: line}
}
