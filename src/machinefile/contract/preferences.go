package contract

import corecontract "github.com/jterrazz/studio-station/src/core/contract"

// value: the preferences group of the shared source and of the
// Machinefile this machine carries.

const GroupSource = "machinefile"

// value: the shared source: a repository read at a pinned commit,
// named as the plan names it (`studio`).

type SharedSource struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Commit string `json:"commit"`
}

type SourcePrefs struct{ p corecontract.Preferences }

func NewSourcePrefs(p corecontract.Preferences) SourcePrefs { return SourcePrefs{p: p} }

type group struct {
	Source *SharedSource `json:"source,omitempty"`
	// value: the Machinefile this machine carries, bound by
	// `station init --machinefile` and `station apply --machinefile`.
	Name string `json:"name,omitempty"`
}

func (s SourcePrefs) Get() (src SharedSource, ok bool, err error) {
	// invariant: a group never written is no shared source: ok is false.
	var g group
	if err := s.p.Read(GroupSource, &g); err != nil || g.Source == nil {
		return SharedSource{}, false, err
	}
	return *g.Source, true, nil
}

func (s SourcePrefs) Bound() (string, error) {
	// value: the Machinefile this machine is bound to; empty when none is.
	var g group
	err := s.p.Read(GroupSource, &g)
	return g.Name, err
}

func (s SourcePrefs) Bind(name string) error {
	// invariant: binding keeps the shared source as it was.
	var g group
	if err := s.p.Read(GroupSource, &g); err != nil {
		return err
	}
	g.Name = name
	return s.p.Write(GroupSource, g)
}
