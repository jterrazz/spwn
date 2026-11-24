package contract

import (
	"regexp"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	core "github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	GroupMacs   = "macs"
	GroupVMUser = "vm_user"
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

const VMUserPattern = `^[a-z]([a-z0-9_.-]{0,30}[a-z0-9_-])?$` // value: the VM user's rule, a name both a macOS short name and a Linux user name take: lowercase, since macOS reads a name whatever its case, a-z first, since a leading _ names macOS's own accounts, then up to 31 of a-z, 0-9, _, . and -, never a dot last.

var vmUserRule = regexp.MustCompile(VMUserPattern)

func CheckVMUser(user string, account bool) error {
	// invariant: a VM user is never the images' own account nor root, which
	// every guest retires or keeps to itself, and holds no newline, since it
	// reaches a guest's scripts as one line. A guest's own accounts are
	// refused by its scripts, by their uid.
	if !vmUserRule.MatchString(user) || user == catalog.UpstreamLogin.User || user == "root" {
		return &machines.BadVMUserError{User: user, Account: account}
	}
	return nil
}

func (v VMUserPrefs) Get() (string, error) {
	var user string
	err := v.p.Read(GroupVMUser, &user)
	return user, err
}

func (v VMUserPrefs) Set(user string) error { return v.p.Write(GroupVMUser, user) }
