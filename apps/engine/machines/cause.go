package machines

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const (
	// value: SPEC B5.5: the lines read at the end of each of a run's two
	// outputs, its console and its hypervisor's own, and the lines its
	// record keeps of them.
	ConsoleWindow = 40
	LastLinesKept = 3
)

func Meaningful(line string) bool {
	// invariant: a blank line, or one that holds only a timestamp, tells
	// nothing of why a run ended.
	return strings.TrimFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r' || r == '[' || r == ']' || r == '.' || r == ':' || r == '-' || r == '+' || r == 'T' || r == 'Z' || (r >= '0' && r <= '9')
	}) != ""
}

func LastLines(lines []string) []string {
	// invariant: the last three lines, unless the line that tells the cause
	// came before them: then that line and the two after it, so the cause
	// is always among the lines kept.
	from := max(0, len(lines)-LastLinesKept)
	if at := causeLine(lines); at >= 0 && at < from {
		from = at
	}
	return append([]string(nil), lines[from:min(len(lines), from+LastLinesKept)]...)
}

func causeLine(lines []string) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if _, ok := catalog.CauseIn(lines[i]); ok {
			return i
		}
	}
	return -1
}

func causeOf(lines []string) (string, bool) {
	if at := causeLine(lines); at >= 0 {
		return catalog.CauseIn(lines[at])
	}
	return "", false
}

func DownCause(end RunEnd) string {
	// invariant: SPEC B5.5: a line of the console's catalogue of causes
	// says why, whatever the run's end; without one, the end's own words.
	if words, ok := causeOf(end.LastLines); ok {
		return words
	}
	switch end.Reason {
	case GuestPoweroff:
		return PoweredOffInside
	case GuestReboot:
		return "rebooted from inside"
	case Crashed:
		return "the hypervisor stopped it"
	case Unknown:
	}
	return "it stopped"
}

func CheckCause(lines []string) string {
	// invariant: a check that fails while the run lives is told by its
	// console when a line of it says why, as a guest's out-of-memory kill
	// of its service does.
	if words, ok := causeOf(lines); ok {
		return words
	}
	return CheckFailed
}
