package settings

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/textblock"
)

func bodyLines(body string) []string { return strings.Split(strings.TrimSuffix(body, "\n"), "\n") }

func PackageBlockIDs(zshrc string) []string {
	// value: the ids of the package blocks ~/.zshrc holds, in file order.
	spans := textblock.Spans(strings.Split(zshrc, "\n"))
	out := make([]string, 0, len(spans))
	for _, s := range spans {
		out = append(out, s.ID)
	}
	return out
}

func UpsertPackageBlock(zshrc, id, body string) (updated string, changed bool) {
	// invariant: an upsert touches the block's own lines alone: a block there is
	// replaced in place; a new one goes right after the last station block, or,
	// when there is none, before the file's last line: its final newline, or a
	// last line of the owner's with no newline. So the owner's lines keep their
	// bytes and their order, a new block's end marker is always followed by a
	// newline, so an installer's `>>` append is never glued to it, and the
	// block's removal gives the file back byte for byte. A station marker that
	// ends the file with no newline gets one, station's own byte, before the
	// new block follows it.
	lines := strings.Split(zshrc, "\n")
	start, end := textblock.Markers(id)
	block := append(append([]string{start}, bodyLines(body)...), end)
	spans := textblock.Spans(lines)
	for _, s := range spans {
		if s.ID == id && s.End >= 0 {
			if have, _ := textblock.Of(zshrc, id); have == body {
				return zshrc, false
			}
			out := append(append(cloned(lines[:s.Start]), block...), lines[s.End+1:]...)
			return strings.Join(out, "\n"), true
		}
	}
	pos, last := len(lines)-1, -1
	if _, end, ok := mainSpan(lines); ok {
		last = end
	}
	for _, s := range spans {
		last = max(last, s.End)
	}
	if last >= 0 {
		pos = last + 1
	}
	if pos == len(lines) {
		lines = append(lines, "")
	}
	out := append(append(cloned(lines[:pos]), block...), lines[pos:]...)
	return strings.Join(out, "\n"), true
}

func cloned(lines []string) []string { return append([]string(nil), lines...) }
