package base

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateWorldID returns an ID like spwn-world-titan-84721.
// When the config is "default", a random planet name is used instead.
// Legacy note: worlds created before v1.1.0 used the "w-" prefix (e.g. w-titan-84721).
// Those IDs are still valid and stored in state.json; only new worlds get the new format.
func GenerateWorldID(configName string) string {
	name := configName
	if name == "default" {
		name = RandomPlanetName()
	}
	return fmt.Sprintf("spwn-world-%s-%s", name, randDigits(5))
}

// GenerateAgentID returns an ID like a-neo-52103.
func GenerateAgentID(agentName string) string {
	return fmt.Sprintf("a-%s-%s", agentName, randDigits(5))
}

func randDigits(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		d, _ := rand.Int(rand.Reader, big.NewInt(10))
		s += fmt.Sprintf("%d", d.Int64())
	}
	return s
}
