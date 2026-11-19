package contract

import (
	"time"

	corecontract "github.com/jterrazz/studio-station/apps/engine/core/contract"
)

// value: the preferences group of the shared source and of the
// Machinefile this machine carries.

const GroupSource = "machinefile"

// value: the shared source: a repository read at its default branch,
// named as the plan names it (`studio`).

type SharedSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type SourcePrefs struct{ p corecontract.Preferences }

func NewSourcePrefs(p corecontract.Preferences) SourcePrefs { return SourcePrefs{p: p} }

type group struct {
	Source *SharedSource `json:"source,omitempty"`
	// value: the Machinefile this machine carries, bound by
	// `station init --machinefile` and `station apply --machinefile`.
	Name string `json:"name,omitempty"`
	// value: when it was bound: the engine's first night is past it.
	BoundAt time.Time `json:"bound_at,omitzero"`
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

func (s SourcePrefs) BoundAt() (time.Time, error) {
	// value: when this machine was bound to its Machinefile; zero when it is
	// not, or was bound by a station that kept no time.
	var g group
	err := s.p.Read(GroupSource, &g)
	return g.BoundAt, err
}

func (s SourcePrefs) Bind(name string, at time.Time) error {
	// invariant: binding keeps the shared source as it was, and binding the
	// same Machinefile again keeps when it was first bound.
	var g group
	if err := s.p.Read(GroupSource, &g); err != nil {
		return err
	}
	if g.Name != name {
		g.Name, g.BoundAt = name, at.UTC()
	}
	return s.p.Write(GroupSource, g)
}
