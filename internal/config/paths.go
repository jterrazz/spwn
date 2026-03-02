package config

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

// BaseDir returns the path to ~/.universe/.
// If UNIVERSE_HOME is set, it overrides the default (used for test isolation).
func BaseDir() string {
	if dir := os.Getenv("UNIVERSE_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, UniverseBaseDir)
}

// UniversesDir returns the path to ~/.universe/universes/.
func UniversesDir() string {
	return filepath.Join(BaseDir(), UniversesSubDir)
}

// AgentsDir returns the path to ~/.universe/agents/.
func AgentsDir() string {
	return filepath.Join(BaseDir(), AgentsSubDir)
}

// StatePath returns the path to ~/.universe/state.json.
func StatePath() string {
	return filepath.Join(BaseDir(), StateFileName)
}

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
