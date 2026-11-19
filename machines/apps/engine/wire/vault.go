package wire

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: the vault a machine keeps station's secrets in: a system's keychain;
// a VM keeps none.
func (m machine) vault() contract.Vault {
	if m.kind != catalog.Mac {
		return noVault{}
	}
	if m.world != nil {
		return m.world.vault()
	}
	return keychain.New(m.sys.Run)
}

type noVault struct{}

func (noVault) Get(context.Context, string) ([]byte, error) { return nil, accounts.ErrNoVault }
func (noVault) Put(context.Context, string, []byte) error   { return accounts.ErrNoVault }
func (noVault) Forget(context.Context, string) error        { return accounts.ErrNoVault }
func (noVault) Names(context.Context) ([]string, error)     { return nil, accounts.ErrNoVault }
