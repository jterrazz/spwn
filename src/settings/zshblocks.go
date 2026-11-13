package settings

import "strings"

const packageMarker = "station:"

func BlockMarkers(id string) (start, end string) {
	// value: the markers of one package block of ~/.zshrc, by its id.
	return "# >>> " + id + " >>>", "# <<< " + id + " <<<"
}

// value: what ~/.zshrc holds of a package block: none, one block, or a
// damage a hand cannot be told from (a start with no end, two of one id).

type BlockState int

const (
	BlockAbsent BlockState = iota
	BlockHere
	BlockDamaged
)

type span struct {
	id         string
	start, end int
}

// invariant: a package block is found by its two marker lines alone, on
// whole lines; the main block and the owner's lines are never part of one.
func spansOf(lines []string) []span {
	var out []span
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		id, ok := strings.CutPrefix(t, "# >>> "+packageMarker)
		if !ok || !strings.HasSuffix(id, " >>>") {
			continue
		}
		id = packageMarker + strings.TrimSuffix(id, " >>>")
		_, end := BlockMarkers(id)
		s := span{id: id, start: i, end: -1}
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == end {
				s.end = j
				break
			}
		}
		out = append(out, s)
		if s.end >= 0 {
			i = s.end
		}
	}
	return out
}

func bodyLines(body string) []string { return strings.Split(strings.TrimSuffix(body, "\n"), "\n") }

func PackageBlock(zshrc, id string) (string, BlockState) {
	// value: the body of one package block, without its markers, and whether
	// it is there.
	lines := strings.Split(zshrc, "\n")
	var found []span
	for _, s := range spansOf(lines) {
		if s.id == id {
			found = append(found, s)
		}
	}
	switch {
	case len(found) == 0:
		return "", BlockAbsent
	case len(found) > 1 || found[0].end < 0:
		return "", BlockDamaged
	}
	s := found[0]
	return strings.Join(lines[s.start+1:s.end], "\n") + "\n", BlockHere
}

func PackageBlockIDs(zshrc string) []string {
	// value: the ids of the package blocks ~/.zshrc holds, in file order.
	spans := spansOf(strings.Split(zshrc, "\n"))
	out := make([]string, 0, len(spans))
	for _, s := range spans {
		out = append(out, s.id)
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
	start, end := BlockMarkers(id)
	block := append(append([]string{start}, bodyLines(body)...), end)
	spans := spansOf(lines)
	for _, s := range spans {
		if s.id == id && s.end >= 0 {
			if have, _ := PackageBlock(zshrc, id); have == body {
				return zshrc, false
			}
			out := append(append(cloned(lines[:s.start]), block...), lines[s.end+1:]...)
			return strings.Join(out, "\n"), true
		}
	}
	pos, last := len(lines)-1, -1
	if _, end, ok := mainSpan(lines); ok {
		last = end
	}
	for _, s := range spans {
		last = max(last, s.end)
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

func RemovePackageBlock(zshrc, id string) (updated string, changed bool) {
	// value: zshrc without the package block of that id, its marker lines
	// and what is between them, every other byte as it was.
	lines := strings.Split(zshrc, "\n")
	for _, s := range spansOf(lines) {
		if s.id == id && s.end >= 0 {
			return strings.Join(append(cloned(lines[:s.start]), lines[s.end+1:]...), "\n"), true
		}
	}
	return zshrc, false
}

func cloned(lines []string) []string { return append([]string(nil), lines...) }
