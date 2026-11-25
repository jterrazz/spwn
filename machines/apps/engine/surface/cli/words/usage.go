package words

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

const (
	UnknownCommand   fail.Code = "usage.unknown-command"
	MissingArgument  fail.Code = "usage.missing-argument"
	EntryWithoutVerb fail.Code = "usage.entry-without-verb"
	NeedsTerminal    fail.Code = "usage.needs-terminal"
)

func usage() map[fail.Code]Wording {
	return map[fail.Code]Wording{
		UnknownCommand: func(e *fail.Error) (string, string) {
			if near := strings.Fields(e.Detail["suggest"]); len(near) > 0 {
				return "unknown command \"" + e.Subject + "\"", Try("", near)
			}
			return "unknown command \"" + e.Subject + "\"", "station help"
		},
		MissingArgument: func(e *fail.Error) (string, string) {
			what := e.Detail["what"]
			return e.Subject + " needs " + article(what) + " " + what, e.Subject + " <" + what + ">"
		},
		EntryWithoutVerb: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is " + article(e.Detail["row"]) + " " + e.Detail["row"] + ": say what to do with it", e.Detail["command"] + " " + e.Subject
		},
		NeedsTerminal: func(e *fail.Error) (string, string) {
			return e.Subject + " needs a terminal", "run it in a terminal"
		},
	}
}

func Try(path string, commands []string) string {
	// value: the commands a refusal offers in place of a word the tree does
	// not know, each one pasteable: `station <path> <word>`.
	prefix := strings.TrimSpace("station " + path)
	said := make([]string, 0, len(commands))
	for _, c := range commands {
		said = append(said, prefix+" "+c)
	}
	return strings.Join(said, ", or ")
}

func article(noun string) string {
	if noun != "" && strings.ContainsRune("aeiou", rune(noun[0])) {
		return "an"
	}
	return "a"
}
