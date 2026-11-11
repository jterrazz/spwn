package contract

import corecontract "github.com/jterrazz/studio-station/src/core/contract"

// value: the preferences group of the shared source.

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

func (s SourcePrefs) Get() (src SharedSource, ok bool, err error) {
	// invariant: a group never written is no shared source: ok is false.
	var group struct {
		Source *SharedSource `json:"source,omitempty"`
	}
	if err := s.p.Read(GroupSource, &group); err != nil || group.Source == nil {
		return SharedSource{}, false, err
	}
	return *group.Source, true, nil
}
