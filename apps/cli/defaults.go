package cli

import (
	"spwn.sh/packages/world"
)

func ensureDefaults() error {
	_ = world.CreateDefaultConfig()
	return nil
}
