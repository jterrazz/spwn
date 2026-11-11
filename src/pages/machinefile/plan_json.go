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
}

type PlanJSON struct {
	Schema      string      `json:"schema"`
	Machinefile string      `json:"machinefile"`
	Needs       []NeedJSON  `json:"needs"`
	Lines       []LineJSON  `json:"lines"`
	Extra       []ExtraJSON `json:"extra"`
	AfterApply  []string    `json:"after_apply"`
	Bound       string      `json:"bound,omitempty"`
}

func lineJSON(l machinefile.Line, result, detail string) LineJSON {
	out := LineJSON{Item: l.Step.Item, Kind: string(l.Step.Kind), Method: l.Step.Method, Condition: string(l.Seen.Condition), Action: string(l.Action), Result: result, Detail: detail}
	for k, v := range l.Step.Values {
		if out.Values == nil {
			out.Values = map[string]string{}
		}
		out.Values[k] = machinefile.Shown(v)
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
		out.Extra = append(out.Extra, ExtraJSON{Item: e.Item, Method: e.Method, Machinefile: e.Machinefile})
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
	for _, d := range append(append([]Worded{}, done...), removed...) {
		detail := d.Detail
		if len(d.Words) > 0 {
			detail = strings.Join(d.Words, ", ")
		}
		out.Lines = append(out.Lines, lineJSON(d.Line, string(d.State), detail))
	}
	return out
}
