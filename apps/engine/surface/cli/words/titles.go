package words

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

func CodesSorted() []fail.Code {
	// value: every code a refusal carries, in one order: the keys of the
	// one table of the codes' words.
	return slices.Sorted(func(yield func(fail.Code) bool) {
		for code := range Codes() {
			if !yield(code) {
				return
			}
		}
	})
}

func Title(code fail.Code) (name, family string) {
	// value: what a code is called where its sentence is not wanted: its
	// name read as words (`too little memory`), and the family it belongs
	// to, the words before its dot (`machines`); a code of the engine's own
	// has a title written by hand, and no family.
	if title, own := engineTitle(code); own {
		return title, ""
	}
	family, slug, dotted := strings.Cut(string(code), ".")
	if !dotted {
		slug, family = family, ""
	}
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, part := range parts {
		switch part {
		case "vm":
			parts[i] = "VM"
		case "mac":
			parts[i] = "Mac"
		}
	}
	return strings.Join(parts, " "), family
}

func engineTitle(code fail.Code) (string, bool) {
	// value: the titles of the engine's own refusals, written by hand: the
	// window shows them as they are, with the glossary's words.
	switch code {
	case enginecalls.StartFailed:
		return "Engine did not start", true
	case enginecalls.VMStopped:
		return "Machine stopped", true
	case enginecalls.UnknownVM:
		return "Unknown machine", true
	case enginecalls.UnknownOp:
		return "Unknown operation", true
	case enginecalls.UnknownPage:
		return "Unknown page", true
	case enginecalls.GuestAbsent:
		return "Guest not connected", true
	case enginecalls.GuestTimeout:
		return "Guest did not answer", true
	case enginecalls.IgnoredPort:
		return "Port never shared", true
	case enginecalls.NotAllowed:
		return "Port outside its rights", true
	case enginecalls.RangeFull:
		return "No free port", true
	case enginecalls.AwakeRun:
		return "Awake cannot run", true
	case enginecalls.AwakeRange:
		return "Awake duration out of range", true
	case enginecalls.AwakeGuard:
		return "Awake held back", true
	case enginecalls.AwakeUnsupported:
		return "Awake not supported", true
	case enginecalls.Internal:
		return "Engine failure", true
	}
	return "", false
}
