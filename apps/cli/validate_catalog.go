package cli

import (
	"sort"
	"strings"

	clidep "spwn.sh/apps/cli/dependency"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/runtimes"
)

func init() {
	clidep.SetCatalogLookup(
		func(ref string) bool {
			for _, t := range dependency.BuiltinTools() {
				if t.Name() == ref {
					return true
				}
			}
			return false
		},
		func() []string {
			out := make([]string, 0, len(dependency.BuiltinTools()))
			for _, t := range dependency.BuiltinTools() {
				out = append(out, strings.TrimPrefix(t.Name(), "spwn:"))
			}
			return out
		},
	)
}

func catalogToolNames() []string {
	deps := dependency.BuiltinTools()
	depNames := make([]string, 0, len(deps))
	for _, t := range deps {
		depNames = append(depNames, t.Name())
	}
	sort.Strings(depNames)

	adapters := runtimes.All()
	rtNames := make([]string, 0, len(adapters))
	for _, a := range adapters {
		if a.Tool != nil {
			rtNames = append(rtNames, a.Tool.Name())
		}
	}
	sort.Strings(rtNames)

	return append(depNames, rtNames...)
}

func supportedRuntimes() []string {
	adapters := runtimes.All()
	out := make([]string, 0, 2*len(adapters))
	for _, a := range adapters {
		out = append(out, a.Name)
		if a.Tool != nil {
			out = append(out, a.Tool.Name())
		}
	}
	return out
}
