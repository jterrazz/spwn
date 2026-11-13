package calls

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

type ShareList struct {
	// invariant: the share list is written by accounts alone; a setting of
	// another family that puts a name on it, or takes one off, does it here.
	prefs contract.ShareListPrefs
}

func NewShareList(prefs contract.ShareListPrefs) ShareList { return ShareList{prefs: prefs} }

func (l ShareList) Get() ([]string, error) { return l.prefs.Get() }

func (l ShareList) Add(name string) error {
	list, err := l.prefs.Get()
	if err != nil || slices.Contains(list, name) {
		return err
	}
	return l.prefs.Set(append(list, name))
}

func (l ShareList) Remove(name string) error {
	list, err := l.prefs.Get()
	if err != nil {
		return err
	}
	return l.prefs.Set(slices.DeleteFunc(list, func(n string) bool { return n == name }))
}
