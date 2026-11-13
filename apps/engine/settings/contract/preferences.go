package contract

import core "github.com/jterrazz/studio-station/apps/engine/core/contract"

const GroupOffHere = "off_here"

type OffHerePrefs struct{ p core.Preferences }

func OffHere(p core.Preferences) OffHerePrefs { return OffHerePrefs{p: p} }

func (o OffHerePrefs) Get() ([]string, error) {
	var names []string
	err := o.p.Read(GroupOffHere, &names)
	return names, err
}

func (o OffHerePrefs) Set(names []string) error { return o.p.Write(GroupOffHere, names) }
