//go:build !linux

package linux

import "github.com/jterrazz/studio-station/apps/engine/core/machine"

func hardware(func(string) ([]byte, error)) (machine.System, machine.Hardware) {
	return machine.System{}, machine.Hardware{}
}
