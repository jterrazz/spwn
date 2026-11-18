package menubar

import "github.com/jterrazz/studio-station/apps/engine/page"

func opened(name string) *page.Value { return hud(name + " opened in Terminal") }

func copied(id, label, words string) page.Action {
	return page.Action{ID: id, Label: label, Icon: "doc.on.doc", On: page.OnPage, Priority: 4, Run: &page.Run{Copy: words}, Hud: hud("Copied " + words)}
}

const (
	keyPorts    = "cmd+p"
	keyAccounts = "cmd+u"
	keyAwake    = "cmd+e"
	durations   = "right-click for a duration"
)
