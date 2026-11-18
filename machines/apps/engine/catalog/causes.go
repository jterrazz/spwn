package catalog

import "strings"

type Cause struct {
	// value: SPEC B5.5: a line a server's console kept, read as why it went
	// down; matched without case, the first cause listed wins a line.
	Pattern, Words string
}

var Causes = []Cause{
	{Pattern: "out of memory: kill", Words: "out of memory"},
	{Pattern: "kernel panic", Words: "kernel panic"},
}

func CauseIn(line string) (string, bool) {
	lower := strings.ToLower(line)
	for _, c := range Causes {
		if strings.Contains(lower, c.Pattern) {
			return c.Words, true
		}
	}
	return "", false
}
