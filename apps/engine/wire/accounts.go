package wire

import (
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/cli"
	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

// value: who each provider says is signed in here, read by its own app, each
// read's process recorded under track.
func (m machine) identities(track string) accountscontract.Identities {
	if m.world != nil {
		return m.world.identities()
	}
	return cli.New(m.sys.Run, m.sys.FS, cli.Options{Home: m.sys.Env.Home(), Track: track})
}
