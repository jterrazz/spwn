package wire

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/cli"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

func (m machine) signIns() contract.SignIns {
	if m.world != nil {
		return m.world.signIns(m.sys.Clock.Now, m.sys.Clock.Sleep)
	}
	return cli.NewSignIns(signInMachines{m})
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

var errUnbuilt = errors.New("the vault and grants are not built into this station yet")

// invariant: until the real adapters land, every call of the vault and the
// grants answers this one error on a machine that is no world.
type (
	unbuiltPasswords struct{}
	unbuiltGrants    struct{}
)

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
func (unbuiltGrants) Candidates(context.Context, string) (accounts.Candidates, error) {
	return accounts.Candidates{}, errUnbuilt
}
