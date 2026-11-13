package secret

import "strings"

func Screen(text string) (kept string, withheld bool) {
	// invariant: each secret in outside words reads Marker, word by word;
	// every other word is kept as given, and withheld says one was replaced.
	kept = privateKey.ReplaceAllLiteralString(text, Marker)
	kept = header.ReplaceAllString(kept, "${1}"+Marker)
	kept = credentials(kept)
	kept = query.ReplaceAllString(kept, "${1}"+Marker)
	kept = issued.ReplaceAllLiteralString(kept, Marker)
	kept = bareRuns(kept)
	return kept, kept != text
}

func Shaped(v string) bool {
	// value: whether a whole value holds a secret, for a value withheld whole
	// rather than word by word.
	_, withheld := Screen(v)
	return withheld
}

func bareRuns(text string) string {
	var b strings.Builder
	last := 0
	for _, at := range run.FindAllStringIndex(text, -1) {
		if !bare(text, at) {
			continue
		}
		b.WriteString(text[last:at[0]])
		b.WriteString(Marker)
		last = at[1]
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}
