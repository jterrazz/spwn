package agents

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func Words(a Agent) []string {
	// invariant: the step line of an agent is its current step, said as it
	// is being done, then its counts, each a part the page joins: "running
	// tests", "41/58 pass"; an agent with no step has no line.
	if len(a.Steps) == 0 {
		return nil
	}
	return stepParts(a.Steps[len(a.Steps)-1], true)
}

func Done(s Step) []string {
	// invariant: a step a later one followed, said as done: "edited
	// src/pages/index.astro", "+86 −12".
	return stepParts(s, false)
}

func stepParts(s Step, current bool) []string {
	said := string(s.Verb)
	for _, w := range catalog.AgentStepsSaid {
		if w.Verb == s.Verb {
			said = w.Done
			if current {
				said = w.Doing
			}
		}
	}
	if s.Object != "" {
		said += " " + s.Object
	}
	parts := []string{said}
	switch {
	case s.Tests != nil:
		parts = append(parts, strconv.Itoa(s.Tests.Pass)+"/"+strconv.Itoa(s.Tests.Total)+" pass")
	case s.Diff != nil:
		parts = append(parts, "+"+strconv.Itoa(s.Diff.Plus)+" −"+strconv.Itoa(s.Diff.Minus))
	}
	return parts
}
