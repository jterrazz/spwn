package wire

import (
	"path/filepath"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/studio"
	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/zshblock"
)

func (m machine) secretSource(o studio.Options) accountscontract.SecretSource {
	if m.world != nil {
		return m.world.secretSource()
	}
	return studio.New(m.sys.Run, o)
}

func (m machine) shell() *zshblock.Block {
	return zshblock.NewBlock(m.sys.FS, catalog.DotfilesFS, m.secretSource(studio.Options{FileBackend: true, Budget: zshblock.TokenBudget}), zshblock.BlockOptions{ThisMachine: m.kind, PortsEnv: filepath.Join(m.paths.Prefs, "services.env")})
}

func (m machine) zshrcPath() string { return filepath.Join(m.sys.Env.Home(), ".zshrc") }
