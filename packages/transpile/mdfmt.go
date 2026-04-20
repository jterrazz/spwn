package transpile

import "strings"

// StripLeadingH1 drops a leading `# …` heading from body, so a
// worldbook-generated block (which has its own H1 by convention)
// can be inlined under a wrapper heading without doubling up.
//
// Used by every renderer that inlines worldbook content: the
// wrapper emits "## Physics" / "## Faculties" / etc. itself and
// then splices the body's content minus its own "# Physics of This
// World" title.
func StripLeadingH1(body string) string {
	body = strings.TrimLeft(body, "\n")
	if !strings.HasPrefix(body, "# ") {
		return body
	}
	if idx := strings.Index(body, "\n"); idx != -1 {
		return strings.TrimLeft(body[idx+1:], "\n")
	}
	return ""
}

// DemoteHeadings prefixes every markdown heading line in body with
// one extra "#". A block whose top-level sections were H2s nests
// cleanly under the H2 wrapper its parent emits.
//
// Code fences (``` …  ```) are preserved verbatim so shell examples
// that happen to start a line with `#` (shell comments, magic
// comments) aren't mistaken for headings and mangled.
func DemoteHeadings(body string) string {
	var out strings.Builder
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		if !inFence && strings.HasPrefix(line, "#") {
			out.WriteByte('#')
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	// strings.Split adds a trailing empty element for bodies ending
	// in \n; trim the extra newline so callers chain cleanly.
	return strings.TrimRight(out.String(), "\n") + "\n"
}
