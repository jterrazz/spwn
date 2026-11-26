package machinefile

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

// value: whether a machine upgrades what its Machinefile lists by itself:
// a Mac bound to the file does, every night, by its engine; the file that
// allows it, none for a Mac bound to no Machinefile; the sentence that says it; and the last night that ran it,
// with its verdict.

type Upgrades struct {
	Nightly  bool
	File     string
	Sentence string
	Last     *LastRun `json:",omitempty"`
}

type LastRun struct {
	At      time.Time
	Verdict verdict.Verdict
}

// value: what a drift reads of a machine's nights: the Machinefile the
// machine is bound to, and its last night, ran false when none ran.

type Nights struct {
	Bound string
	Night Night
	Ran   bool
}

func UpgradesOf(machine, file string, mac bool, n Nights) Upgrades {
	// invariant: only a Mac's engine runs a night, and only for the
	// Machinefile the Mac is bound to; a VM moves at a station machinefile update run
	// inside it. The last night is said only when it ran that file.
	switch {
	case !mac:
		return Upgrades{File: file, Sentence: machine + " upgrades only when you ask."}
	case n.Bound == "":
		return Upgrades{Sentence: "Never by itself: " + machine + " is bound to no Machinefile."}
	case n.Bound != file:
		return Upgrades{File: n.Bound, Sentence: "Never by this file: " + machine + " upgrades " + n.Bound + " every night."}
	}
	u := Upgrades{Nightly: true, File: file, Sentence: "Every night at 03:00, as " + file + " allows."}
	if n.Ran && n.Night.Machinefile == file {
		u.Last = &LastRun{At: n.Night.At, Verdict: n.Night.verdict()}
	}
	return u
}

func (n Night) verdict() verdict.Verdict {
	// value: the last night in the owner's words: what stopped it, what
	// failed, or what it moved, then what it left waiting for the owner.
	switch {
	case n.Error != "":
		return verdict.Verdict{Tone: verdict.Bad, Sentence: "It did not run to its end.", Line: n.Error}
	case n.Interrupted:
		return verdict.Verdict{Tone: verdict.Warn, Sentence: "It was interrupted.", Line: "It runs again at the engine's next start."}
	case len(n.Failed) > 0:
		return verdict.Verdict{Tone: verdict.Bad, Sentence: counted(len(n.Failed), "update") + " failed.", Line: moved(len(n.Moved))}
	}
	v := verdict.Verdict{Tone: verdict.OK, Sentence: moved(len(n.Moved))}
	if waits := len(n.Waiting) + len(n.Left) + len(n.Deferred); waits > 0 || n.Unpulled != "" {
		v.Tone = verdict.Warn
		v.Line = counted(waits, "app") + " waits for you."
		if waits != 1 {
			v.Line = counted(waits, "app") + " wait for you."
		}
		if n.Unpulled != "" {
			v.Line = "Its shared source was not pulled: " + n.Unpulled
		}
	}
	return v
}

func moved(n int) string {
	if n == 0 {
		return "Nothing moved."
	}
	return counted(n, "app") + " moved."
}

func counted(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}
