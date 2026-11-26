package wire

import (
	"github.com/spf13/cobra"

	automationsreceipts "github.com/jterrazz/studio-station/apps/engine/automations/adapter/receipts"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract"
	automationscmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/automations"
	automationslog "github.com/jterrazz/studio-station/apps/engine/surface/cli/automations/log"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func (m machine) automations() contract.Automations {
	if m.world != nil {
		return m.world.automations()
	}
	return automationsreceipts.Automations{}
}

func automationsCommand(env kit.Env, m machine) *cobra.Command {
	project := m.agentsProject
	return automationscmd.Command(automationscmd.Deps{ThisMachine: m.where, Project: project}, env,
		automationslog.Command(automationslog.Deps{ThisMachine: m.where, Project: project}, env),
	)
}
