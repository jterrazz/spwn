package foundation

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateUniverseID returns an ID like u-default-84721.
func GenerateUniverseID(configName string) string {
	return fmt.Sprintf("u-%s-%s", configName, randDigits(5))
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
