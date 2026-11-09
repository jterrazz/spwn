package menubar

import (
	"strings"

	"github.com/jterrazz/studio-station/src/page"
)

const ActionSharePortSheet = "share-port:"

// invariant: SPEC A5 Share a port: one sheet per workspace, Of swapping
// between them; under it what listens there and is not shared yet, each
// with Share, the call its row runs; nothing listening, the line that
// says so and no button. Nothing is typed. The sheet the page opens is
// its first.
func sharePortSheets(listeners []page.Item, to []workspace, self string) []page.Action {
	if len(to) == 1 {
		return []page.Action{sharePortSheet(ActionShare, to[0], nil, listeners, self)}
	}
	out := []page.Action{sharePortSheet(ActionShare, to[0], to, listeners, self)}
	for _, t := range to {
		out = append(out, sharePortSheet(ActionSharePortSheet+t.VM, t, to, listeners, self))
	}
	return out
}

func sharePortSheet(id string, t workspace, every []workspace, listeners []page.Item, self string) page.Action {
	of := page.Field{Key: "Of", Label: "Of", Kind: page.FieldSelect, Swaps: true, Default: id}
	if len(every) == 0 {
		of.Options = []page.Option{workspaceOption(id, t, self)}
	}
	for _, o := range every {
		of.Options = append(of.Options, workspaceOption(ActionSharePortSheet+o.VM, o, self))
	}
	if len(every) > 0 {
		of.Default = ActionSharePortSheet + t.VM
	}
	rows := page.Field{Key: "Port", Label: "Listening now in " + t.VM, Kind: page.FieldRows, Note: "Nothing listens in " + t.VM + " right now."}
	if t.VM == "" {
		of.Options, rows.Label, rows.Note = nil, "Listening now", "No workspace runs to share a port of."
	}
	for _, it := range listeners {
		r := it.Row
		if r == nil || r.Verb == "" || !strings.HasPrefix(r.ID, t.VM+":") {
			continue
		}
		rows.Options = append(rows.Options, page.Option{Value: r.Verb, Label: r.Label, Note: r.ID, Glyph: r.Glyph})
	}
	form := &page.Form{Title: "Share a port", Subtitle: "Private on your tailnet. Pick a listener, no typing.", Fields: []page.Field{of, rows}}
	return page.Action{ID: id, Label: "Share a port", On: page.OnPage, Priority: 3, Mutates: true, Sheet: page.SheetSharePort, Form: form}
}

func workspaceOption(value string, t workspace, self string) page.Option {
	return page.Option{Value: value, Label: t.VM, Note: "on " + page.MacName(t.Mac, self), Glyph: page.Glyph{Kind: "workspace"}}
}

// value: the workspaces a page of ports offers to share from: the page's
// own VM, the running workspaces of its Mac, or of every Mac; read none,
// those whose listeners it lists.
func shareTargets(to []workspace, listeners []page.Item, sc portsScope) []workspace {
	var out []workspace
	for _, t := range to {
		if sc.VM != "" && t.VM != sc.VM || sc.Mac != "" && t.Mac != sc.Mac {
			continue
		}
		out = append(out, t)
	}
	if len(out) > 0 || len(to) > 0 {
		return out
	}
	for _, it := range listeners {
		if it.Row == nil {
			continue
		}
		vm := it.Row.ID[:strings.LastIndex(it.Row.ID, ":")]
		if !containsVM(out, vm) {
			out = append(out, workspace{VM: vm})
		}
	}
	return out
}

func containsVM(ws []workspace, vm string) bool {
	for _, w := range ws {
		if w.VM == vm {
			return true
		}
	}
	return false
}
