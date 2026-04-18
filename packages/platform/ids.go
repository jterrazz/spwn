package platform

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// GenerateWorldID returns a world ID of the shape
// world-<slug>-<5-hex-digits>. The hash prevents name collisions
// on concurrent spawns.
func GenerateWorldID(name string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("world-%s-%02x%02x%x", slugify(name), b[0], b[1], b[2]&0xf)
}

// GenerateAgentID returns an agent ID of the shape agent-<name>-<5-hex>.
func GenerateAgentID(name string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("agent-%s-%02x%02x%x", name, b[0], b[1], b[2]&0xf)
}

// RandomPlanetName picks a random planet/moon for a world.
func RandomPlanetName() string { return PlanetNames[randInt(len(PlanetNames))] }

// RandomAgentName picks a random agent name.
func RandomAgentName() string { return AgentNames[randInt(len(AgentNames))] }

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prev := byte('-')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteByte(c)
			prev = c
		} else if prev != '-' {
			b.WriteByte('-')
			prev = '-'
		}
	}
	return strings.Trim(b.String(), "-")
}

func randInt(n int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(nBig.Int64())
}
