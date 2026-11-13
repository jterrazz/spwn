package machines

import (
	"slices"
	"strconv"
)

func FirstFree(image string, free func(string) bool) string {
	// value: the name a new VM of image takes unasked: <image>-<n>, the
	// first n that free allows.
	for n := 1; ; n++ {
		if name := image + "-" + strconv.Itoa(n); free(name) {
			return name
		}
	}
}

func Held(f AllMachines) []string {
	// value: every name the fleet as read holds: each Mac's, each VM's and
	// each agent a VM keeps a record of, sorted.
	var out []string
	for _, h := range f.Macs {
		out = append(out, h.Name)
		for _, vm := range h.VMs {
			out = append(out, vm.Name)
			out = append(out, vm.Agents...)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
