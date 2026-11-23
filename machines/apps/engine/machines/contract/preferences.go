package contract

import (
	"regexp"
	"strings"

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

// value: a user name every guest takes as it is: POSIX's portable short
// name, lowercase, so no system that folds case reads it as another.
var shortName = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

func (v VMUserPrefs) Get() (string, error) {
	// invariant: a vm_user set is a short name and never root: it reaches a
	// guest's scripts as one line, and macOS reads "Root" as root.
	var user string
	if err := v.p.Read(GroupVMUser, &user); err != nil || user == "" {
		return user, err
	}
	if !shortName.MatchString(user) || strings.EqualFold(user, "root") {
		return "", &machines.BadVMUserError{User: user}
	}
	return user, nil
}

func (v VMUserPrefs) Set(user string) error { return v.p.Write(GroupVMUser, user) }
