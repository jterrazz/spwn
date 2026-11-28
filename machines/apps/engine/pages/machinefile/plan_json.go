package machinefile

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

type LineJSON struct {
	Package   string `json:"package"`
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
	Need     string   `json:"need"`
	Packages []string `json:"packages"`
}

type ExtraJSON struct {
	Package     string `json:"package"`
	Method      string `json:"method"`
	Machinefile string `json:"machinefile"`
	// value: what the package reads now, and what `apply --remove` does with
	// it: `remove` (an extra already gone only leaves the ledger) or
	// `leave` (present another way since, needed by another app there, or
	// not readable now).
	Condition string `json:"condition"`
	Action    string `json:"action"`
}

// value: one block of ~/.zshrc: its package and id, what the file holds of it
// (`present`, `absent`, `outdated`, `changed`, or `extra` for one station
// wrote for an extra), what apply does with it (`set`, `keep`, `leave`,
// `remove`), and what a run did (`ok`, `broken`...); a plan's block has no
// result, and a run lists only the blocks it judged, so an extra's block is
// in a run with `--remove` alone.

type BlockJSON struct {
	Package   string `json:"package"`
	ID        string `json:"id"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Result    string `json:"result,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// value: one Machinefile's lines of the plan's phase, that Machinefile,
// and once their turn came how they ended (`absent` when not run after a
// file that failed, `broken` when the phase's time was spent before it).

type RunJSON struct {
	Phase       string   `json:"phase"`
	Machinefile string   `json:"machinefile"`
	Lines       []string `json:"lines"`
	Result      string   `json:"result,omitempty"`
	Detail      string   `json:"detail,omitempty"`
}

// value: one line of the accounts the chain gives: its key, the programs
// one file gives under it, and that file; or that file's clearing of it.

type AccountJSON struct {
	Key         string   `json:"key"`
	Programs    []string `json:"programs"`
	Machinefile string   `json:"machinefile"`
	Cleared     bool     `json:"cleared,omitempty"`
}

func runJSON(s machinefile.Script) RunJSON {
	return RunJSON{Phase: string(s.Phase), Machinefile: s.Origin.Name, Lines: machinefile.ScriptLines(s.Lines)}
}

func ranJSON(ran []machinefile.Ran) []RunJSON {
	out := make([]RunJSON, 0, len(ran))
	for _, r := range ran {
		j := runJSON(r.Script)
		j.Result, j.Detail = string(r.State), r.Detail
		out = append(out, j)
	}
	return out
}

type PlanJSON struct {
	Schema      string        `json:"schema"`
	Machinefile string        `json:"machinefile"`
	Needs       []NeedJSON    `json:"needs"`
	Lines       []LineJSON    `json:"lines"`
	Extra       []ExtraJSON   `json:"extra"`
	Blocks      []BlockJSON   `json:"blocks,omitempty"`
	AfterApply  []string      `json:"after_apply"`
	Run         []RunJSON     `json:"run,omitempty"`
	Bound       string        `json:"bound,omitempty"`
	Accounts    []AccountJSON `json:"accounts,omitempty"`
}

func lineJSON(l machinefile.Line, result, detail string) LineJSON {
	out := LineJSON{Package: l.Step.Package, Kind: string(l.Step.Kind), Method: l.Method(), Condition: string(l.Seen.Condition), Action: string(l.Action), Result: result, Detail: detail}
	for k, v := range l.Step.Values {
		if out.Values == nil {
			out.Values = map[string]string{}
		}
		out.Values[k] = string(v)
	}
	return out
}

func PlanContract(schema string, p machinefile.Plan) PlanJSON {
	// value: station.machinefile.plan/v3 and station.machinefile.drift/v4:
	// the plan's needs, its lines, what is extra, what follows an apply,
	// each file's lines of its phase, and the accounts the chain gives,
	// one line per key and file.
	out := PlanJSON{Schema: schema, Machinefile: p.Name, Needs: []NeedJSON{}, Lines: []LineJSON{}, Extra: []ExtraJSON{}, AfterApply: []string{}}
	for _, a := range p.Needs {
		out.Needs = append(out.Needs, NeedJSON{Need: string(a.Need), Packages: a.Packages})
	}
	for _, l := range p.Lines {
		out.Lines = append(out.Lines, lineJSON(l, "", l.Seen.Detail))
	}
	for _, e := range p.Extra {
		out.Extra = append(out.Extra, ExtraJSON{Package: e.Step.Package, Method: e.Step.Method, Machinefile: e.Installed.Machinefile, Condition: string(e.Seen.Condition), Action: string(e.Removal())})
	}
	for _, b := range p.Blocks {
		out.Blocks = append(out.Blocks, BlockJSON{Package: b.Package, ID: b.ID, Condition: string(b.Condition), Action: string(b.Action)})
	}
	for _, r := range p.Restarts {
		out.AfterApply = append(out.AfterApply, string(r))
	}
	for _, s := range p.PhaseScripts() {
		out.Run = append(out.Run, runJSON(s))
	}
	for _, a := range p.Accounts {
		out.Accounts = append(out.Accounts, AccountJSON{Key: a.Key, Programs: append([]string{}, a.Programs...), Machinefile: a.By.Name, Cleared: a.Cleared})
	}
	return out
}

func AppliedContract(schema string, a machinefile.Applied, done, removed []Worded) PlanJSON {
	// value: station.machinefile.apply/v3: the plan's shape, each line with
	// what the run did with it, then each extra removed, and each block the
	// run judged with what it did.
	out := PlanContract(schema, a.Plan)
	out.Lines, out.Bound, out.Run = []LineJSON{}, a.Bound, ranJSON(a.Ran)
	blocks := out.Blocks
	out.Blocks = nil
	for _, j := range blocks {
		for _, b := range a.Blocks {
			if b.ID == j.ID {
				j.Result, j.Detail = string(b.State), b.Detail
				out.Blocks = append(out.Blocks, j)
			}
		}
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
