package contract

import (
	core "github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	GroupMacs = "macs"
	// value: the name station v3.0 and earlier kept the other Macs under.
	RetiredGroupMacs = "hosts"
	GroupVMUser      = "vm_user"
)

type MacsPrefs struct{ p core.Preferences }

func Macs(p core.Preferences) MacsPrefs { return MacsPrefs{p: p} }

func (h MacsPrefs) Get() (map[string]machines.MacRecord, error) {
	var macs map[string]machines.MacRecord
	err := h.p.Read(GroupMacs, &macs)
	return macs, err
}

func (h MacsPrefs) Set(macs map[string]machines.MacRecord) error { return h.p.Write(GroupMacs, macs) }

type VMUserPrefs struct{ p core.Preferences }

func VMUser(p core.Preferences) VMUserPrefs { return VMUserPrefs{p: p} }

func (v VMUserPrefs) Get() (string, error) {
	var user string
	err := v.p.Read(GroupVMUser, &user)
	return user, err
}

func (v VMUserPrefs) Set(user string) error { return v.p.Write(GroupVMUser, user) }
