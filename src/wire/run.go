package wire

import (
	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	settingscmd "github.com/jterrazz/studio-station/src/surface/cli/settings"
)

func runCommand(env kit.Env, m machine) *cobra.Command {
	return settingscmd.Run(env, settingscmd.RunGit(env, m.gitShortcuts()), settingscmd.RunContainers(env, m.where))
}

// value: the aliases of the Git shortcuts setting, as the catalog holds them.
func (m machine) gitShortcuts() []catalog.GitAlias {
	it, _ := m.cat.Setting(catalog.GitShortcutsSetting)
	aliases, _ := it.Effect.(catalog.GitAliases)
	return aliases.Aliases
}
