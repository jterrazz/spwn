package foundation

import "testing"

func TestRandomCosmosWord_NonEmpty(t *testing.T) {
	word := RandomCosmosWord()
	if word == "" {
		t.Error("RandomCosmosWord() returned empty string")
	}
}

func TestRandomCosmosWord_InList(t *testing.T) {
	valid := make(map[string]bool)
	for _, w := range CosmosWords {
		valid[w] = true
	}

	for i := 0; i < 100; i++ {
		word := RandomCosmosWord()
		if !valid[word] {
			t.Errorf("RandomCosmosWord() = %q, not in CosmosWords list", word)
		}
	}
}

func TestRandomCosmosWord_Randomness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		seen[RandomCosmosWord()] = true
	}
	// With 20 words and 100 calls, we should see at least 2 distinct words
	if len(seen) < 2 {
		t.Errorf("RandomCosmosWord() returned same word in 100 calls - randomness likely broken")
	}
}

func TestRandomAgentName_NonEmpty(t *testing.T) {
	name := RandomAgentName()
	if name == "" {
		t.Error("RandomAgentName() returned empty string")
	}
}

func TestRandomAgentName_InList(t *testing.T) {
	valid := make(map[string]bool)
	for _, n := range AgentNames {
		valid[n] = true
	}

	for i := 0; i < 100; i++ {
		name := RandomAgentName()
		if !valid[name] {
			t.Errorf("RandomAgentName() = %q, not in AgentNames list", name)
		}
	}
}

func TestRandomAgentName_Randomness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		seen[RandomAgentName()] = true
	}
	if len(seen) < 2 {
		t.Errorf("RandomAgentName() returned same name in 100 calls - randomness likely broken")
	}
}
