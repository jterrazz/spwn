package machinefile

import (
	"strings"

	"github.com/jterrazz/studio-station/src/machinefile"
)

type LineJSON struct {
	Item      string `json:"item"`
	Kind      string `json:"kind"`
	Method    string `json:"method"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
	// value: what the run did with it (`ok`, `broken`...); a plan's line has none.
	Result string            `json:"result,omitempty"`
	Values map[string]string `json:"values,omitempty"`
	Detail string            `json:"detail,omitempty"`
}

type NeedJSON struct {
	Need  string   `json:"need"`
	Items []string `json:"items"`
}

type ExtraJSON struct {
	Item        string `json:"item"`
	Method      string `json:"method"`
	Machinefile string `json:"machinefile"`
	// value: what the item reads now, and what `apply --remove` does with
	// it: `remove` (an extra already gone only leaves the ledger) or
	// `leave` (present another way since, needed by another app there, or
	// not readable now).
	Condition string `json:"condition"`
	Action    string `json:"action"`
}

// value: one block of ~/.zshrc: its item and id, what the file holds of it
// (`present`, `absent`, `changed`, or `extra` for one whose item is no longer
// listed), what apply does with it (`set`, `keep`, `leave`, `remove`), and
// what a run did (`ok`, `broken`...); a plan's block has no result.

type BlockJSON struct {
	Item      string `json:"item"`
	ID        string `json:"id"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Result    string `json:"result,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type PlanJSON struct {
	Schema      string      `json:"schema"`
	Machinefile string      `json:"machinefile"`
	Needs       []NeedJSON  `json:"needs"`
	Lines       []LineJSON  `json:"lines"`
	Extra       []ExtraJSON `json:"extra"`
	Blocks      []BlockJSON `json:"blocks,omitempty"`
	AfterApply  []string    `json:"after_apply"`
	Bound       string      `json:"bound,omitempty"`
}

func lineJSON(l machinefile.Line, result, detail string) LineJSON {
	out := LineJSON{Item: l.Step.Item, Kind: string(l.Step.Kind), Method: l.Method(), Condition: string(l.Seen.Condition), Action: string(l.Action), Result: result, Detail: detail}
	for k, v := range l.Step.Values {
		if out.Values == nil {
			out.Values = map[string]string{}
		}
		out.Values[k] = string(v)
	}
	return out
}

func PlanContract(schema string, p machinefile.Plan) PlanJSON {
	// value: station.machinefile.plan/v1 and station.machinefile.drift/v1:
	// the plan's needs, its lines, what is extra and what follows an apply.
	out := PlanJSON{Schema: schema, Machinefile: p.Name, Needs: []NeedJSON{}, Lines: []LineJSON{}, Extra: []ExtraJSON{}, AfterApply: []string{}}
	for _, a := range p.Needs {
		out.Needs = append(out.Needs, NeedJSON{Need: string(a.Need), Items: a.Items})
	}
	for _, l := range p.Lines {
		out.Lines = append(out.Lines, lineJSON(l, "", l.Seen.Detail))
	}
	for _, e := range p.Extra {
		out.Extra = append(out.Extra, ExtraJSON{Item: e.Step.Item, Method: e.Step.Method, Machinefile: e.Installed.Machinefile, Condition: string(e.Seen.Condition), Action: string(e.Action)})
	}
	for _, b := range p.Blocks {
		out.Blocks = append(out.Blocks, BlockJSON{Item: b.Item, ID: b.ID, Condition: string(b.Condition), Action: string(b.Action)})
	}
	for _, r := range p.Restarts {
		out.AfterApply = append(out.AfterApply, string(r))
	}
	return out
}

func AppliedContract(a machinefile.Applied, done, removed []Worded) PlanJSON {
	// value: station.machinefile.apply/v1: the plan's shape, each line with
	// what the run did with it, then each extra removed.
	out := PlanContract("station.machinefile.apply/v1", a.Plan)
	out.Lines, out.Bound = []LineJSON{}, a.Bound
	for i, b := range a.Blocks {
		out.Blocks[i].Result, out.Blocks[i].Detail = string(b.State), b.Detail
	}
	for _, d := range append(append([]Worded{}, done...), removed...) {
		detail := d.Detail
		if len(d.Words) > 0 {
			detail = strings.Join(d.Words, ", ")
		}
		out.Lines = append(out.Lines, lineJSON(d.Line, string(d.State), detail))
	}
	return out
}
