package agents

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var Nth = string(rune(0xB7))

var agentName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}(` + Nth + `[0-9]{1,3})?$`)

func ValidName(name string) bool {
	// invariant: SPEC B5.22: a name is lowercase letters, digits and -, then
	// at most one ·<n>.
	return agentName.MatchString(name)
}

func NameFor(vm string, taken []string) string {
	// invariant: a VM's first agent takes the VM's name, the next ones
	// <vm>·<n> from 2, the first of them whose key no agent holds.
	if !Clashes(taken, vm) {
		return vm
	}
	for n := 2; ; n++ {
		if name := vm + Nth + strconv.Itoa(n); !Clashes(taken, name) {
			return name
		}
	}
}

func Named(kept []Agent, vm string) []Agent {
	// invariant: every agent of a VM has a name and its VM: one no start
	// named, an agent the owner ran by hand, is given the next free name of
	// the VM in the order it began.
	out := slices.Clone(kept)
	var taken []string
	for _, a := range out {
		if a.Name != "" {
			taken = append(taken, a.Name)
		}
	}
	order := make([]int, 0, len(out))
	for i := range out {
		out[i].VM = cmp.Or(out[i].VM, vm)
		if out[i].Name == "" {
			order = append(order, i)
		}
	}
	slices.SortStableFunc(order, func(i, j int) int {
		return cmp.Or(compareTime(out[i].Since, out[j].Since), strings.Compare(out[i].Session, out[j].Session))
	})
	for _, i := range order {
		out[i].Name = NameFor(vm, taken)
		taken = append(taken, out[i].Name)
	}
	return out
}

func compareTime(a, b time.Time) int {
	switch {
	case a.IsZero() && !b.IsZero():
		return 1
	case b.IsZero() && !a.IsZero():
		return -1
	}
	return a.Compare(b)
}

func Live(a Agent) bool {
	// value: an agent whose terminal still runs: anything but finished.
	return a.State != Finished
}

func SessionOf(name string) string {
	// value: the journal's session of an agent station started, until its
	// app's hook names its own: its name in a session's letters.
	return "station-" + Key(name)
}

func Key(name string) string {
	// value: the name in a session's and a Herdr terminal's letters, where ·
	// is -.
	return strings.ReplaceAll(name, Nth, "-")
}

func Clashes(taken []string, name string) bool {
	// invariant: SPEC B5.22: a name is taken when another's key is its own,
	// so linux-1·2 and linux-1-2 never share a journal entry or a terminal.
	return slices.ContainsFunc(taken, func(t string) bool { return Key(t) == Key(name) })
}
