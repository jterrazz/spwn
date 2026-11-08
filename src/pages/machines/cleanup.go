package machines

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

const cleanupEmpty = "Nothing finished to delete on "

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

func CleanupConfirm(cands []model.Candidate) page.Confirm {
	// value: SPEC B5.9: the y/n of a cleanup names every throwaway it
	// deletes and lists what goes with each one.
	lines := make([]page.Value, 0, len(cands))
	for _, c := range cands {
		lines = append(lines, page.Texts(append([]string{c.Name}, append(idle(c), "goes with "+losses(c))...)...))
	}
	return page.Confirm{Question: "Delete " + throwaways(cands) + "?", Lines: lines, Danger: true, Yes: "Delete"}
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
	return page.Text("refusing to delete " + throwaways(cands) + " without --yes: " + strings.Join(parts, "; "))
}

func idle(c model.Candidate) []string {
	// value: why a candidate may go, and since when: "stopped", or
	// "finished 12 min ago", then how in its agent's words.
	if c.Stopped && c.Ended.IsZero() {
		return []string{"stopped"}
	}
	return []string{"finished " + ago(c.Idle), c.Reason}
}

func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + " min ago"
	case d < 48*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + " hr ago"
	}
	return strconv.Itoa(int(d/(24*time.Hour))) + " days ago"
}

func losses(c model.Candidate) string {
	// value: SPEC A5: what a throwaway's delete takes with it, "its disk
	// (12 GB)", "2 shared ports", "2 shared accounts", joined with "and".
	disk := "its disk"
	if c.DiskBytes > 0 {
		disk += " (" + DiskSize(c.DiskBytes).Text + ")"
	}
	parts := []string{disk}
	if n := len(c.Ports); n > 0 {
		parts = append(parts, counted(n, "shared port"))
	}
	if n := len(c.Accounts); n > 0 {
		parts = append(parts, counted(n, "shared account"))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func throwaways(cands []model.Candidate) string {
	if len(cands) == 1 {
		return cands[0].Name
	}
	return strconv.Itoa(len(cands)) + " throwaways"
}

func counted(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}
