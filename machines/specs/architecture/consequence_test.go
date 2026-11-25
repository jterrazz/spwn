package architecture_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

// invariant: ADR-097's table, pinned here apart from core/confirm: what
// each action can lose at worst, on a machine that runs, that someone holds
// and that hosts a server. The code may not grade an action below it, nor
// above it.
func consequences() map[confirm.Action]confirm.Consequence {
	return map[confirm.Action]confirm.Consequence{
		confirm.Install: confirm.Reversible, confirm.Share: confirm.Reversible, confirm.Start: confirm.Reversible,
		confirm.Restart: confirm.Reversible, confirm.Edit: confirm.Reversible, confirm.StopSharing: confirm.Reversible,
		confirm.Undeclare: confirm.Reversible, confirm.BuildImage: confirm.Reversible, confirm.Relocate: confirm.Reversible,
		confirm.Stop: confirm.StopsOthers, confirm.OffHere: confirm.StopsOthers, confirm.RestartServer: confirm.StopsOthers,
		confirm.Remove: confirm.StopsOthers, confirm.SignOut: confirm.StopsOthers, confirm.RemovePassword: confirm.StopsOthers,
		confirm.RemoveGrant: confirm.StopsOthers, confirm.Overwrite: confirm.StopsOthers, confirm.Forget: confirm.StopsOthers,
		confirm.DeleteImage: confirm.StopsOthers, confirm.RestartMac: confirm.StopsOthers, confirm.StopEngine: confirm.StopsOthers,
		confirm.Cleanup: confirm.StopsOthers, confirm.FreeUnused: confirm.StopsOthers,
		confirm.Delete: confirm.Destroys, confirm.Erase: confirm.Destroys,
	}
}

// value: the verbs of a call that destroys or stops what it names, so it
// declares the confirmation that grades it.
func confirmedVerbs() []string {
	return []string{"delete", "erase", "remove", "forget", "stop", "restart-mac", "logout", "free-unused", "cleanup"}
}

type grading func(a confirm.Action, k machine.Kind, facts ...confirm.Fact) confirm.Consequence

var worst = []confirm.Fact{confirm.Running, confirm.Held, confirm.HostsServer}

func order(c confirm.Consequence) int {
	return slices.Index([]confirm.Consequence{confirm.Reversible, confirm.StopsOthers, confirm.Destroys}, c)
}

// invariant: rule confirmation-follows-consequence (ADR-097): every action
// is graded, at worst, exactly as the pinned table says, each consequence
// asks its one level, and every call whose verb destroys or stops declares
// a confirmation.
func ruleConfirmationFollowsConsequence(grade grading, metas []call.Meta) []finding {
	var out []finding
	pinned := consequences()
	for _, a := range confirm.Actions() {
		want, ok := pinned[a]
		if !ok {
			out = append(out, finding{"confirmation-follows-consequence", string(a), "an action of no row of the table"})
			continue
		}
		got := confirm.Reversible
		for _, k := range []machine.Kind{machine.Computer, machine.Server, machine.Workspace} {
			if g := grade(a, k, worst...); order(g) > order(got) {
				got = g
			}
		}
		switch {
		case order(got) < order(want):
			out = append(out, finding{"confirmation-follows-consequence", string(a), "graded " + string(got) + ", weaker than its consequence, " + string(want)})
		case order(got) > order(want):
			out = append(out, finding{"confirmation-follows-consequence", string(a), "graded " + string(got) + ", above its consequence, " + string(want)})
		}
	}
	for c, l := range map[confirm.Consequence]confirm.Level{confirm.Reversible: confirm.None, confirm.StopsOthers: confirm.YesNo, confirm.Destroys: confirm.TypeName} {
		if c.Level() != l {
			out = append(out, finding{"confirmation-follows-consequence", string(c), "asks " + string(c.Level()) + ", not " + string(l)})
		}
	}
	for _, m := range metas {
		_, verb, _ := strings.Cut(m.Name, ".")
		if slices.Contains(confirmedVerbs(), verb) && m.Confirm == "" {
			out = append(out, finding{"confirmation-follows-consequence", m.Name, "a call that destroys or stops declares no confirmation"})
		}
	}
	return out
}

func TestEveryConfirmationFollowsItsConsequence(t *testing.T) {
	t.Parallel()
	metas := registeredMetas(t)
	t.Run("holds", func(t *testing.T) { excuse(t, ruleConfirmationFollowsConsequence(confirm.ConsequenceOf, metas), nil) })
	t.Run("bites", func(t *testing.T) {
		mutant := func(a confirm.Action, k machine.Kind, facts ...confirm.Fact) confirm.Consequence {
			if a == confirm.Erase {
				return confirm.Reversible
			}
			if a == confirm.Stop {
				return confirm.Destroys
			}
			return confirm.ConsequenceOf(a, k, facts...)
		}
		unconfirmed := append(slices.Clone(metas), call.Meta{Name: "disk.erase"})
		found := ruleConfirmationFollowsConsequence(mutant, unconfirmed)
		bites(t, "confirmation-follows-consequence", found, string(confirm.Erase))
		bites(t, "confirmation-follows-consequence", found, string(confirm.Stop))
		bites(t, "confirmation-follows-consequence", found, "disk.erase")
	})
}
