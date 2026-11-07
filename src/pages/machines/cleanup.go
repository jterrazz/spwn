package machines

import (
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

const (
	CleanupGoes  = "Each one goes with its disk, its ports and its shared accounts."
	cleanupEmpty = "Nothing finished to delete on "
)

func Cleaned(out model.Cleaned, mac string, thisMac page.Context) page.Page {
	// value: SPEC B5.9: each throwaway deleted, then what stays named.
	rows := make([]page.Row, 0, len(out.Deleted))
	for _, name := range out.Deleted {
		rows = append(rows, done(name, catalog.StateOK, "deleted", "with its disk, its ports and its shared accounts"))
	}
	var extra []page.Section
	if words := Stays(out.Kept); words.Text != "" {
		extra = append(extra, page.Section{ID: "kept", Role: page.Banner, Uncounted: true, Items: []page.Item{{Note: &page.Note{Text: words.Text}}}})
	}
	p := pages.Outcome("machines cleanup", thisMac, nil, rows, "VMs", extra...)
	if len(rows) == 0 {
		p.Sections[0].Items = []page.Item{{Empty: &page.Empty{Text: cleanupEmpty + page.MacName(mac, thisMac.Machine)}}}
	}
	return p
}

func Stays(kept []string) page.Value {
	// value: the note of a cleanup: every VM still running, which stays.
	switch len(kept) {
	case 0:
		return page.None()
	case 1:
		return page.Text(kept[0] + " is still running, so it stays.")
	}
	return page.Text(strings.Join(kept, ", ") + " are still running, so they stay.")
}
