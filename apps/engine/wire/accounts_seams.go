package wire

import (
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
	return m.vaultKept(m.macEngine()).Passwords()
}

func (m machine) grants() contract.Grants {
	if m.world != nil {
		return m.world.grants(m.sys.Clock.Now, m.sys.Clock.Sleep)
	}
	return m.vaultKept(m.macEngine()).Grants()
}
