package wordlist

import (
	"crypto/rand"
	"math/big"
)

// ConfigWords are cosmos-themed words for universe config names.
var ConfigWords = []string{
	"nebula", "quasar", "pulsar", "zenith", "epoch",
	"nova", "cosmos", "orbit", "photon", "prism",
	"aurora", "vertex", "helix", "cipher", "flux",
	"vector", "nexus", "apex", "vortex", "horizon",
	"solstice", "equinox", "eclipse", "comet", "meteor",
	"astral", "stellar", "lunar", "solar", "radiant",
	"plasma", "fusion", "fission", "proton", "neutron",
	"boson", "quark", "lepton", "muon", "tau",
	"gamma", "delta", "sigma", "omega", "lambda",
	"titan", "atlas", "orion", "lyra", "vega",
	"altair", "rigel", "sirius", "polaris", "castor",
	"calypso", "triton", "oberon", "ariel", "miranda",
	"pandora", "helios", "selene", "chronos", "aether",
}

// AgentWords are human first names and mythological names for agents.
var AgentWords = []string{
	"leonardo", "aurora", "felix", "iris", "cyrus",
	"nova", "orion", "luna", "atlas", "cleo",
	"dante", "freya", "hugo", "ivy", "jasper",
	"kai", "lara", "miles", "nora", "oscar",
	"petra", "quinn", "rhea", "soren", "thea",
	"una", "viggo", "wren", "xander", "yara",
	"zara", "arlo", "bria", "cato", "dara",
	"elio", "fern", "gaia", "hana", "idris",
	"juno", "kira", "leo", "mira", "nero",
	"opal", "pax", "rune", "sage", "tara",
	"ulric", "veda", "wade", "xena", "yuri",
	"zeno", "abel", "beau", "cora", "dion",
}

// PickConfig returns a random cosmos-themed word.
func PickConfig() string {
	return pick(ConfigWords)
}

// PickAgent returns a random agent name.
func PickAgent() string {
	return pick(AgentWords)
}

func pick(words []string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
	return words[n.Int64()]
}
