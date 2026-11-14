package wire

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

func (m machine) signIns() contract.SignIns {
	if m.world != nil {
		return m.world.signIns(m.sys.Clock.Now, m.sys.Clock.Sleep)
	}
	return unbuiltSignIns{}
}

func (m machine) passwords() contract.Passwords {
	if m.world != nil {
		return m.world.passwords(m.sys.Clock.Now, m.sys.Clock.Sleep)
	}
	return unbuiltPasswords{}
}

func (m machine) grants() contract.Grants {
	if m.world != nil {
		return m.world.grants(m.sys.Clock.Now, m.sys.Clock.Sleep)
	}
	return unbuiltGrants{}
}

var errUnbuilt = errors.New("signing in a machine, the vault and grants are not built into this station yet")

// invariant: until the real adapters land, every call of the new accounts
// shapes answers this one error on a machine that is no world.
type (
	unbuiltSignIns   struct{}
	unbuiltPasswords struct{}
	unbuiltGrants    struct{}
)

func (unbuiltSignIns) Read(context.Context, string) (accounts.ProgramsRead, error) {
	return accounts.ProgramsRead{}, errUnbuilt
}
func (unbuiltSignIns) Login(context.Context, accounts.Program) (<-chan contract.LoginUpdate, error) {
	return nil, errUnbuilt
}
func (unbuiltSignIns) CheckSignOut(context.Context, accounts.Program) error { return errUnbuilt }
func (unbuiltSignIns) SignOut(context.Context, accounts.Program) (accounts.ProgramRow, error) {
	return accounts.ProgramRow{}, errUnbuilt
}
func (unbuiltSignIns) Open(context.Context, string, string) (string, error) { return "", errUnbuilt }
func (unbuiltSignIns) Answer(context.Context, string, string, secret.Sent) error {
	return errUnbuilt
}

func (unbuiltPasswords) List(context.Context) (accounts.VaultRead, error) {
	return accounts.VaultRead{}, errUnbuilt
}
func (unbuiltPasswords) Check(context.Context, string) error { return errUnbuilt }
func (unbuiltPasswords) Add(context.Context, contract.PasswordIn) (string, error) {
	return "", errUnbuilt
}
func (unbuiltPasswords) Remove(context.Context, string) ([]accounts.VaultGrant, error) {
	return nil, errUnbuilt
}

func (unbuiltGrants) Check(context.Context, string, accounts.GrantTo, string) error {
	return errUnbuilt
}
func (unbuiltGrants) Add(context.Context, contract.GrantIn) (accounts.Granted, error) {
	return accounts.Granted{}, errUnbuilt
}
func (unbuiltGrants) Remove(context.Context, string, accounts.GrantTo, string) (accounts.Revoked, error) {
	return accounts.Revoked{}, errUnbuilt
}
