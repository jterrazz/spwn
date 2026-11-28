package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestAWorldMachineKeepsItsSpwnSkill(t *testing.T) {
	contracttest.Skill(t, func(*testing.T) contract.Skill {
		return fake.SkillOf(&world.World{ThisMac: "here"}, "here")
	})
}
