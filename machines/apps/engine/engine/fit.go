package engine

type FitVerdict string

const (
	Fits        FitVerdict = "fits"
	EngineOlder FitVerdict = "engine-older"
	AppOlder    FitVerdict = "app-older"
)

type Fit struct {
	Verdict FitVerdict `json:"verdict"`
	// value: the revision the engine speaks, and the one the client declared
	// it draws; zero is a client that declared none.
	Engine int `json:"engine"`
	App    int `json:"app"`
}

func FitOf(engine, app int) Fit {
	// invariant: ADR-072: before 1.0 there is no range: only the one
	// revision the client draws fits; a client that declares none is older
	// than the verdict.
	f := Fit{Engine: engine, App: app, Verdict: Fits}
	switch {
	case engine < app:
		f.Verdict = EngineOlder
	case engine > app:
		f.Verdict = AppOlder
	}
	return f
}
