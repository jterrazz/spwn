package agents

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

func Scrub(text string) (title, detail string) {
	// invariant: SPEC B5.1's privacy rules, before anything is kept: the
	// first line is the title and at most one sentence after it the detail,
	// each cut at a word to catalog.AgentLineCap; a fenced block or a line
	// shaped like code says catalog.AgentCodeSaid; an inline code span keeps
	// its text, without its backticks, only when it holds a path or a name;
	// a word shaped like a secret is dropped.
	lines := said(text)
	if len(lines) == 0 {
		return "", ""
	}
	title = capped(lines[0])
	if len(lines) > 1 {
		detail = capped(firstSentence(strings.Join(lines[1:], " ")))
	}
	if detail == title {
		detail = ""
	}
	return title, detail
}

func said(text string) []string {
	var out []string
	fenced := false
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			if !fenced {
				out = appendCode(out)
			}
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if codeShaped(line) {
			out = appendCode(out)
			continue
		}
		line = scrubbed(spans(line))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func appendCode(lines []string) []string {
	if len(lines) > 0 && lines[len(lines)-1] == catalog.AgentCodeSaid {
		return lines
	}
	return append(lines, catalog.AgentCodeSaid)
}

var (
	spanShape = regexp.MustCompile("`([^`]*)`")
	nameShape = regexp.MustCompile(`^[\w./@#:+~-]+$`)
)

func spans(line string) string {
	return spanShape.ReplaceAllStringFunc(line, func(span string) string {
		inner := strings.Trim(span, "`")
		if len([]rune(inner)) <= catalog.AgentNameCap && nameShape.MatchString(inner) {
			return inner
		}
		return ""
	})
}

func scrubbed(line string) string {
	words := strings.Fields(line)
	kept := words[:0]
	for _, w := range words {
		if !secret.Shaped(strings.Trim(w, ".,;:!?()\"'")) {
			kept = append(kept, w)
		}
	}
	return strings.Join(kept, " ")
}

func codeShaped(line string) bool {
	// invariant: a line is code when its symbols and keywords outweigh its
	// words: at least four signals, and as many as it has words.
	if line == "" {
		return false
	}
	signals := 0
	for _, r := range line {
		if strings.ContainsRune("{}[]();=<>|&$\\", r) {
			signals++
		}
	}
	words := strings.FieldsFunc(line, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' })
	for _, w := range words {
		switch w {
		case "func", "return", "const", "let", "var", "import", "def", "class", "fn", "pub", "package", "elif", "lambda", "async", "await", "struct", "impl", "typeof", "nil", "null", "undefined":
			signals += 2
		}
	}
	return signals >= 4 && signals >= len(strings.Fields(line))
}

func firstSentence(text string) string {
	for i, r := range text {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		if i+1 == len(text) || text[i+1] == ' ' {
			return strings.TrimSpace(text[:i+1])
		}
	}
	return strings.TrimSpace(text)
}

func capped(s string) string {
	runes := []rune(s)
	if len(runes) <= catalog.AgentLineCap {
		return s
	}
	// invariant: a line is cut at its last space within the cap, the word
	// past it dropped whole; a line with no space there is cut at the cap.
	cut := string(runes[:catalog.AgentLineCap+1])
	if i := strings.LastIndex(cut, " "); i > 0 {
		return strings.TrimRight(cut[:i], " ,;:")
	}
	return string(runes[:catalog.AgentLineCap])
}
