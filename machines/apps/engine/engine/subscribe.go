package engine

const (
	PageKeepAwake = "keep-awake"
	PageMenubar   = "menubar"
	// value: the menu bar's pages of SPEC §10.2, each watched by its name
	// and, after a colon, its subject: `menubar.machine:infra`, that
	// machine's page; `menubar.ports:mac-mini`, the ports of that
	// Mac; `menubar.activity:6m`, This Mac over 6 min, and
	// `menubar.activity:infra:24h`, infra over 24 h.
	PageMachine  = "menubar.machine"
	PagePorts    = "menubar.ports"
	PageAwake    = "menubar.awake"
	PageActivity = "menubar.activity"
	PageAccounts = "menubar.accounts"
	PageNeeds    = "menubar.needs"
	PageFind     = "menubar.find"
)

type Opened struct {
	Stream string   `json:"stream"`
	Pages  []string `json:"pages,omitempty"`
	// value: the revision of `station.page/v1` the pages of a watch of pages
	// speak, so a client judges the engine before any page of it.
	Page int `json:"page,omitempty"`
	// value: the verdict on the revision the client declared; a watch whose
	// verdict is not Fits sends nothing after its opening.
	Fit *Fit `json:"fit,omitempty"`
}
