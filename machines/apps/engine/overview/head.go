package overview

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func HeadOf(s Summary) *verdict.Verdict {
	// invariant: a Mac's own head: its engine stopped, else online with the
	// machines it runs, its line how long it is up, empty where that is not
	// known: machines.list's sentence of the Mac is the window's header,
	// never this line. None on a VM, whose Mac says its head.
	if !s.Head.Mac() {
		return nil
	}
	v := macHead(s)
	if v.Tone == verdict.OK && s.Head.Up > 0 {
		v.Line = "Up " + machines.Silence(s.Head.Up) + "."
	}
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
