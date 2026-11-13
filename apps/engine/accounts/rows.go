package accounts

const (
	OriginOwn     = "own"
	OriginShared  = "shared"
	OriginBoth    = "own+shared"
	OriginUnknown = "?"

	EngineRunning  = "running"
	EngineStopped  = "stopped"
	EngineAbsent   = "not installed"
	GuestConnected = "connected"
	GuestAbsent    = "absent"
)

type VM struct {
	// value: a VM of this host as its record and the daemon's list say.
	Name, Lifetime string
	Running        bool
	Shares         []Loan
}

type VMRead struct {
	Name     string    `json:"name"`
	Guest    string    `json:"guest"`
	Accounts []Account `json:"accounts"`
}

func RedactAccount(a Account) Account {
	scrub := func(v string) string {
		c, redacted := Redact(v)
		a.Redacted = a.Redacted || redacted
		return c
	}
	scrubAll := func(vs []string) []string {
		out := make([]string, 0, len(vs))
		for _, v := range vs {
			out = append(out, scrub(v))
		}
		return out
	}
	a.Identity, a.Scopes = scrubAll(a.Identity), scrubAll(a.Scopes)
	a.Source, a.Detail, a.Notes = scrub(a.Source), scrub(a.Detail), scrubAll(a.Notes)
	if a.Shared != nil {
		l := *a.Shared
		l.Name = scrub(l.Name)
		a.Shared = &l
	}
	return a
}
