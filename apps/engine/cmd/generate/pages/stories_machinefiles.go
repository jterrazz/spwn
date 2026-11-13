package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const (
	baseFile  = "jterrazz-base"
	devFile   = "linux-dev"
	exposedVM = "linux-1"
)

// value: the owner's Machinefiles as the play world keeps them on This
// Mac, by path under its station files: a base every machine extends, the
// Mac's own, the workspace every dev VM is made from, and linux-1's, which
// narrows it with the ports its Mac may open to the internet.
func ownerFiles() map[string]string {
	file := func(name string) string { return "machinefiles/" + name + catalog.MachinefileSuffix }
	return map[string]string{
		file(baseFile):  "machinefile: station.machinefile/v1\nname: " + baseFile + "\nmachine: { kind: workspace }\napps: [jq, gh]\n",
		file(proName):   "machinefile: station.machinefile/v1\nname: " + proName + "\nmachine: { kind: computer, system: { os: macos } }\nextends: [" + baseFile + "]\napps: [lazygit]\n",
		file(devFile):   "machinefile: station.machinefile/v1\nname: " + devFile + "\nmachine: { kind: workspace, system: { os: ubuntu, version: '24.04' } }\nextends: [" + baseFile + "]\napps: [ripgrep]\n",
		file(exposedVM): "machinefile: station.machinefile/v1\nname: " + exposedVM + "\nmachine: { kind: workspace }\nextends: [" + devFile + "]\nports: { open: [3000], to: public }\n",
	}
}

// value: the dev VMs made from linux-dev, with what apt holds in each:
// agent-7f is what its file says, linux-1 lacks ripgrep and holds gh a
// version behind; false for a VM made from nothing.
func devApps(vm string) (*playApps, bool) {
	switch vm {
	case "agent-7f":
		return &playApps{Installed: map[string]string{"jq": "1.7.1", "gh": "2.45.0", "ripgrep": "14.1.0"}}, true
	case "linux-1":
		return &playApps{Installed: map[string]string{"jq": "1.7.1", "gh": "2.45.0"}, Behind: map[string]string{"gh": "2.46.0"}}, true
	}
	return nil, false
}

// value: a dev VM's own copy of the owner's Machinefiles it was made
// from, which it reads for its drift, and the one it is bound to.
func devFiles() map[string]string {
	files := ownerFiles()
	delete(files, "machinefiles/"+proName+catalog.MachinefileSuffix)
	delete(files, "machinefiles/"+exposedVM+catalog.MachinefileSuffix)
	return files
}
