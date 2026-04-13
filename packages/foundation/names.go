package foundation

import (
	"crypto/rand"
	"math/big"
)

// CosmosWords are cosmos-themed names for world configs.
var CosmosWords = []string{
	"nebula", "quasar", "pulsar", "nova", "corona",
	"aurora", "zenith", "vortex", "prism", "helix",
	"photon", "neutron", "proton", "boson", "meson",
	"cosmos", "astral", "stellar", "orbit", "vertex",
}

// PlanetNames are planet and celestial body names for worlds.
var PlanetNames = []string{
	"mercury", "venus", "terra", "mars", "jupiter",
	"saturn", "uranus", "neptune", "pluto", "ceres",
	"titan", "europa", "ganymede", "callisto", "io",
	"enceladus", "triton", "oberon", "ariel", "miranda",
	"pandora", "rhea", "dione", "tethys", "hyperion",
	"charon", "sedna", "eris", "haumea", "makemake",
}

// AgentNames are curated names for agents.
var AgentNames = []string{
	"neo", "aurora", "felix", "atlas", "iris",
	"orion", "luna", "nova", "sage", "echo",
	"aria", "theo", "zara", "kai", "lyra",
	"dante", "cleo", "niko", "mira", "juno",
}

// RandomCosmosWord picks a random cosmos-themed word.
func RandomCosmosWord() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(CosmosWords))))
	return CosmosWords[n.Int64()]
}

// RandomPlanetName picks a random planet/moon name for a world.
func RandomPlanetName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(PlanetNames))))
	return PlanetNames[n.Int64()]
}

// RandomAgentName picks a random agent name.
func RandomAgentName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(AgentNames))))
	return AgentNames[n.Int64()]
}
