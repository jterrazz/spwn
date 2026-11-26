package cli

import (
	"strings"

	clidep "spwn.sh/apps/cli/dependency"
	"spwn.sh/packages/dependency/builtin"
	"spwn.sh/packages/runtimes"
)

func init() {
	clidep.SetCatalogLookup(
		func(ref string) bool {
			for _, t := range builtin.BuiltinTools() {
				if t.Name() == ref {
					return true
				}
			}
			return false
		},
		func() []string {
			out := make([]string, 0, len(builtin.BuiltinTools()))
			for _, t := range builtin.BuiltinTools() {
				out = append(out, strings.TrimPrefix(t.Name(), "spwn:"))
			}
			return out
		},
	)
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
