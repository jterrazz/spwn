package settings

import (
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

type Report struct {
	Package catalog.Setting
	State   catalog.State
	Detail  string
	Applies bool
}

type Change struct {
	Package string
	Verb    string
	Result  catalog.State
	Before  []KeyValue
	After   []KeyValue
	Steps   []ChangeStep
}

type KeyValue struct{ Key, Value string }

func Answered(changes []Change) []Change {
	// invariant: a change leaves its call as an answer, screened once and
	// only here: each step's detail (a tool's words, its argv) and each value
	// read before and after, word by word, before any surface or another Mac
	// reads it.
	if changes == nil {
		return nil
	}
	out := make([]Change, len(changes))
	for i, c := range changes {
		c.Before, c.After = keptValues(c.Before), keptValues(c.After)
		if c.Steps != nil {
			steps := make([]ChangeStep, len(c.Steps))
			for j, st := range c.Steps {
				st.Detail = secret.Kept(st.Detail)
				steps[j] = st
			}
			c.Steps = steps
		}
		out[i] = c
	}
	return out
}

func (o Observation) Answered() Observation {
	// invariant: what a setting reads now leaves its call as an answer
	// screened, as a change does: its file's diff and each value before and
	// after, word by word.
	o.Diff = secret.Kept(o.Diff)
	o.Before, o.After = keptValues(o.Before), keptValues(o.After)
	return o
}

func keptValues(in []KeyValue) []KeyValue {
	if in == nil {
		return nil
	}
	out := make([]KeyValue, len(in))
	for i, kv := range in {
		out[i] = KeyValue{Key: kv.Key, Value: secret.Kept(kv.Value)}
	}
	return out
}

type ChangeStep struct {
	Label  string
	State  catalog.State
	Detail string
}

type StudioMissingError struct{ Package string }

func (e *StudioMissingError) Error() string {
	return e.Package + " is kept by the studio CLI, which is not on PATH"
}

func Sets(it catalog.Setting) []KeyValue {
	// value: the values station sets for a setting that writes no file.
	out := make([]KeyValue, len(it.Sets))
	for i, v := range it.Sets {
		out[i] = KeyValue{Key: v.Key, Value: v.Value}
	}
	return out
}
