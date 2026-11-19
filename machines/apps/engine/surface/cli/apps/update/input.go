package appsupdate

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Input struct {
	Names       []string
	Manager     []string
	AllMachines bool
}

func binder(managers []string) func(in *Input, args []string) error {
	return func(in *Input, args []string) error {
		in.Names = args
		if len(in.Names) > 0 && len(in.Manager) > 0 {
			return kit.Refuse("name apps or give --manager, not both", "station apps update --manager "+in.Manager[0])
		}
		for _, v := range in.Manager {
			if !slices.Contains(managers, v) {
				return kit.Refuse("unknown manager \""+v+"\"", "--manager takes "+strings.Join(managers, ", "))
			}
		}
		return nil
	}
}
