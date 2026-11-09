package engine

import (
	"context"

	"github.com/jterrazz/studio-station/src/awake"
)

const (
	PageKeepAwake = "keep-awake"
	PageMenubar   = "menubar"
	// value: the menu bar's pages of SPEC §10.2, each watched by its name
	// and, after a colon, its subject: `menubar.machine:infra`, that
	// machine's page; `menubar.ports:jterrazz-mac-mini`, the ports of that
	// Mac; `menubar.activity:6m`, This Mac over 6 min, and
	// `menubar.activity:infra:24h`, infra over 24 h.
	PageMachine  = "menubar.machine"
	PagePorts    = "menubar.ports"
	PageAwake    = "menubar.awake"
	PageActivity = "menubar.activity"
	PageAccounts = "menubar.accounts"
	PageNeeds    = "menubar.needs"
	PageFind     = "menubar.find"
	PageAgent    = "menubar.agent"
)

type FeedInput struct {
	// value: the daemon's own state moved (VMs, guests, leases, Power).
	Changed <-chan struct{}
	Refresh <-chan struct{}
	Awake   func() awake.View
	// value: what a page is of, after its page's name (`menubar.machine:infra`,
	// `menubar.ports:jterrazz-mac-mini`); empty for every other page.
	Subject string
}

type Feed func(ctx context.Context, in FeedInput) <-chan any

type Opened struct {
	Stream string   `json:"stream"`
	Pages  []string `json:"pages,omitempty"`
}
