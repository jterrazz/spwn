package contract

import (
	core "github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

const (
	GroupName      = "name"
	GroupUserEmail = "user_email"
	GroupUserName  = "user_name"
)

const (
	// value: alias of the key station v2 kept this machine's name under,
	// read under name and written there at the next write.
	RetiredGroupName = "here"
)

type NamePrefs struct{ p core.Preferences }

func Name(p core.Preferences) NamePrefs { return NamePrefs{p: p} }

func (n NamePrefs) Get() (string, error) {
	var name string
	err := n.p.Read(GroupName, &name)
	return name, err
}

func (n NamePrefs) Set(name string) error { return n.p.Write(GroupName, name) }

type IdentityPrefs struct{ p core.Preferences }

func Identity(p core.Preferences) IdentityPrefs { return IdentityPrefs{p: p} }

func (i IdentityPrefs) Get() (setup.Identity, error) {
	var id setup.Identity
	if err := i.p.Read(GroupUserEmail, &id.UserEmail); err != nil {
		return setup.Identity{}, err
	}
	err := i.p.Read(GroupUserName, &id.UserName)
	return id, err
}

func (i IdentityPrefs) Set(id setup.Identity) error {
	if err := i.p.Write(GroupUserEmail, id.UserEmail); err != nil {
		return err
	}
	return i.p.Write(GroupUserName, id.UserName)
}
