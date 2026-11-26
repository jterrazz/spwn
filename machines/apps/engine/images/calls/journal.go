package calls

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/images"
)

// invariant: a build keeps one line: the images it made, or, failed, the
// ones it was asked for, else the ones it made before it failed; a build
// that names none and made none keeps none.
func keptBuild(in BuildIn, out []images.Record, err error) []journal.Line {
	names := in.Names
	if len(out) > 0 && (err == nil || len(names) == 0) {
		names = nil
		for _, r := range out {
			names = append(names, r.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	text := strings.Join(names, ", ") + " built"
	if err != nil {
		text = strings.Join(names, ", ") + " not built"
	}
	line := journal.Line{Text: text}
	if len(names) == 1 {
		line.Subject = &journal.Subject{Kind: journal.Image, Name: names[0]}
	}
	return []journal.Line{line}
}
