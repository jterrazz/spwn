package contract

import core "github.com/jterrazz/studio-station/apps/engine/core/contract"

const GroupNeverSleeps = "never_sleeps"

type NeverSleepsPrefs struct{ p core.Preferences }

func NeverSleeps(p core.Preferences) NeverSleepsPrefs { return NeverSleepsPrefs{p: p} }

func (n NeverSleepsPrefs) Get() (bool, error) {
	var on bool
	err := n.p.Read(GroupNeverSleeps, &on)
	return on, err
}

func (n NeverSleepsPrefs) Set(on bool) error {
	// invariant: off removes the group, so a Mac that never applied it and
	// one that turned it off read alike.
	return n.p.Write(GroupNeverSleeps, on)
}
