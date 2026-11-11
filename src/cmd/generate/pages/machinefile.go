package pages

import (
	"github.com/jterrazz/studio-station/src/machinefile"
	pagemachinefile "github.com/jterrazz/studio-station/src/pages/machinefile"
)

func explained() []machinefile.Value {
	linux := machinefile.Origin{Name: "station-linux", Layer: machinefile.Builtin, File: "built-in/machinefiles/station-linux.yaml"}
	mine := machinefile.Origin{Name: "studio-linux", Layer: machinefile.User, File: "~/.config/station/machinefiles/studio-linux.yaml"}
	return []machinefile.Value{
		{Key: "machine.kind", Text: "workspace", Origin: linux},
		{Key: "machine.system.os", Text: "ubuntu", Origin: linux},
		{Key: "vm.resources.cpus", Text: "8", Origin: mine},
		{Key: "apps.jq", Text: "listed", Origin: mine},
		{Key: "apps.jq.method", Text: "github-release", Origin: machinefile.Origin{Name: "jq", Layer: machinefile.User, File: "~/.config/station/catalog/command-line/jq.yaml"}},
		{Key: "apps.git", Text: "optional, no method for ubuntu 24.04", Origin: linux},
	}
}

func machinefilePages() []Fixture {
	return states("machinefile",
		pending(pagemachinefile.Explain("studio-linux", "", explained(), thisMacContext())),
		pagemachinefile.Explain("studio-linux", "", nil, thisMacContext()),
		failure("machinefile", "explain", "no Machinefile named jterrazz-macbook-pro", "station explain --machinefile <name>"),
		pagemachinefile.Explain("studio-linux", "", explained(), thisMacContext()))
}
