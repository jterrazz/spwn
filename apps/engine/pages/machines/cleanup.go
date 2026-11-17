package machines

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

const cleanupEmpty = "Nothing stopped to delete on "

func Cleaned(out model.Cleaned, mac string, thisMac page.Context) page.Page {
	// value: SPEC B5.9: each ephemeral workspace deleted, then what stays named.
	rows := make([]page.Row, 0, len(out.Deleted))
	for _, name := range out.Deleted {
		rows = append(rows, done(name, catalog.StateOK, "deleted", "with its disk, its ports and its grants"))
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

func CleanupConfirm(cands []model.Candidate) page.Confirm {
	// value: SPEC B5.9: the y/n of a cleanup names every ephemeral workspace it
	// deletes and lists what goes with each one.
	lines := make([]page.Value, 0, len(cands))
	for _, c := range cands {
		lines = append(lines, page.Texts(c.Name, "stopped", "goes with "+losses(c)))
	}
	return page.Confirm{Question: "Delete " + ephemerals(cands) + "?", Lines: lines, Danger: true, Yes: "Delete"}
}

func CleanupRefused(cands []model.Candidate) page.Value {
	// value: a cleanup refused off a terminal says who would have gone, and
	// with what.
	if len(cands) == 1 {
		return page.Text("refusing to delete " + cands[0].Name + " without --yes: it goes with " + losses(cands[0]))
	}
	parts := make([]string, 0, len(cands))
	for _, c := range cands {
		parts = append(parts, c.Name+" goes with "+losses(c))
	}
	return page.Text("refusing to delete " + ephemerals(cands) + " without --yes: " + strings.Join(parts, "; "))
}

func losses(c model.Candidate) string {
	// value: SPEC A5: what an ephemeral workspace's delete takes with it, "its disk
	// (12 GB)", "2 shared ports", "2 grants", joined with "and".
	disk := "its disk"
	if c.DiskBytes > 0 {
		disk += " (" + DiskSize(c.DiskBytes).Text + ")"
	}
	parts := []string{disk}
	if n := len(c.Ports); n > 0 {
		parts = append(parts, counted(n, "shared port"))
	}
	if n := len(c.Accounts); n > 0 {
		parts = append(parts, counted(n, "grant"))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func ephemerals(cands []model.Candidate) string {
	if len(cands) == 1 {
		return cands[0].Name
	}
	return strconv.Itoa(len(cands)) + " ephemeral workspaces"
}

func counted(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}
