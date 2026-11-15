package overview

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

func HeadOf(s Summary) *verdict.Verdict {
	// invariant: a Mac's own head: its engine stopped, else online with the
	// machines it runs; none on a VM, whose Mac says its head.
	if !s.Head.Mac() {
		return nil
	}
	v := macHead(s)
	return &v
}

func macHead(s Summary) verdict.Verdict {
	switch {
	case s.Engine == EngineStopped:
		return verdict.Verdict{Tone: verdict.Bad, Sentence: SentenceEngineStopped, Line: LineEngineStopped}
	case s.VMs == nil:
		return verdict.Verdict{Tone: verdict.OK, Sentence: "Online."}
	}
	running := 0
	for _, vm := range s.VMs {
		if vm.Running {
			running++
		}
	}
	switch running {
	case 0:
		return verdict.Verdict{Tone: verdict.OK, Sentence: "Online, with no machine running."}
	case 1:
		return verdict.Verdict{Tone: verdict.OK, Sentence: "Online, with 1 machine running."}
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: "Online, with " + strconv.Itoa(running) + " machines running."}
}

func (s Summary) Told(sentence string) Summary {
	// value: the Mac's sentence, as machines.list tells it, set as the line
	// of its head while the engine runs.
	if s.Head.Verdict != nil && s.Engine != EngineStopped {
		v := *s.Head.Verdict
		v.Line = sentence
		s.Head.Verdict = &v
	}
	return s
}
