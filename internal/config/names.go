package config

import (
	"crypto/rand"
	"math/big"
)

// CosmosWords are cosmos-themed names for universe configs.
var CosmosWords = []string{
	"nebula", "quasar", "pulsar", "nova", "corona",
	"aurora", "zenith", "vortex", "prism", "helix",
	"photon", "neutron", "proton", "boson", "meson",
	"cosmos", "astral", "stellar", "orbit", "vertex",
}

// AgentNames are curated names for agents.
var AgentNames = []string{
	"leonardo", "aurora", "felix", "atlas", "iris",
	"orion", "luna", "nova", "sage", "echo",
	"aria", "theo", "zara", "kai", "lyra",
	"dante", "cleo", "niko", "mira", "juno",
}

// RandomCosmosWord picks a random cosmos-themed word.
func RandomCosmosWord() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(CosmosWords))))
	return CosmosWords[n.Int64()]
}

// RandomAgentName picks a random agent name.
func RandomAgentName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(AgentNames))))
	return AgentNames[n.Int64()]
}
