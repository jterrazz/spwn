package menubar

import (
	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"strconv"
	"strings"
	"time"
)

// value: the accounts This Mac shared with the workspace, each with what
// it may do, its time left and its Take back.
func sharedSection(m Machine, h model.MacView, vm model.VM) page.Section {
	from := model.Label(*m.All, h)
	s := page.Section{ID: SectionShared, Title: "Accounts shared from " + from, Columns: columns()}
	for _, a := range sharedAccounts(m, vm.Name) {
		left := a.Shared.Until.Sub(m.Now)
		words := make([]page.Value, 0, len(a.Scopes)+1)
		for _, scope := range a.Scopes {
			if !strings.Contains(scope, ":") {
				words = append(words, page.Text(scope))
			}
		}
		row := page.Row{ID: a.Provider, Label: accounts.Label(a.Provider), Mark: page.OK, Cells: []page.Value{page.List(append(words, page.Text(hours(left)+" left"))...)}}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No account shared"}}}
	}
	return s
}

func sharedAccounts(m Machine, vm string) []accounts.Account {
	if m.Accounts == nil {
		return nil
	}
	var out []accounts.Account
	for _, mc := range m.Accounts.Machines {
		if mc.Name != vm {
			continue
		}
		for _, a := range mc.Accounts {
			if a.Shared != nil {
				out = append(out, a)
			}
		}
	}
	return out
}

func hours(d time.Duration) string {
	if d < time.Hour {
		return strconv.Itoa(max(int(d/time.Minute), 1)) + " min"
	}
	return strconv.Itoa(int(d/time.Hour)) + " h"
}

func takeBacks(m Machine, vm model.VM) []page.Action {
	shared := sharedAccounts(m, vm.Name)
	out := make([]page.Action, 0, len(shared))
	for _, a := range shared {
		label := accounts.Label(a.Provider)
		c := ask(m.Asks, "take-back", machine.Workspace, label, "Take back "+label+" from "+vm.Name+"?", "take back", "Its copy on "+vm.Name+" is deleted.")
		out = append(out, page.Action{ID: ActionTakeBack + "/" + a.Provider, Label: "Take back", On: page.OnRow, Rows: []string{a.Provider}, Priority: 4, Mutates: true, Confirm: c,
			Run: &page.Run{Call: CallTakeBack, In: answered(c, map[string]string{"Account": a.Provider, "From": vm.Name})}})
	}
	return out
}

// value: a Mac's Accounts segment: the accounts it shared with its VMs, each with
// its time left.
func macAccounts(m Machine, h model.MacView) page.Section {
	s := page.Section{ID: SectionAccounts, Title: "Accounts", Segment: SegmentAccounts, Columns: columns()}
	for _, vm := range h.VMs {
		for _, a := range sharedAccounts(m, vm.Name) {
			row := page.Row{ID: vm.Name + "/" + a.Provider, Label: accounts.Label(a.Provider), Mark: page.OK,
				Cells: []page.Value{page.Texts("shared with "+vm.Name, hours(a.Shared.Until.Sub(m.Now))+" left")}}
			s.Items = append(s.Items, page.Item{Row: &row})
		}
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No account shared"}}}
	}
	return s
}
