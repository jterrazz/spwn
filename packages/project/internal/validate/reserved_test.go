package validate

import "testing"

func TestReservedAgentNames_followTheCommandsThatExist(t *testing.T) {
	for _, name := range []string{"agent", "world", "install", "up"} {
		if !IsReservedAgentName(name) {
			t.Errorf("%q is a command of spwn, so it is reserved", name)
		}
	}
	for _, name := range []string{"check", "skill"} {
		if IsReservedAgentName(name) {
			t.Errorf("%q is no command of spwn any more, so an agent may take it", name)
		}
	}
}
