package wire

import (
	"path/filepath"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/zshblock"
)

func (m machine) shell() *zshblock.Block {
	return zshblock.NewBlock(m.sys.FS, catalog.DotfilesFS, zshblock.BlockOptions{ThisMachine: m.kind, PortsEnv: filepath.Join(m.paths.Prefs, "services.env")})
}

func (m machine) zshrcPath() string { return filepath.Join(m.sys.Env.Home(), ".zshrc") }
