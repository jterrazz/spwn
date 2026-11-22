package words

import "github.com/jterrazz/studio-station/apps/engine/catalog"

func SettingsAsRoot(verb, pkg string) string {
	return catalog.StationAsRoot("settings " + verb + " " + pkg)
}

func PasswordlessRestart(mac string) string {
	return "check that sudo needs no password for fdesetup on " + mac
}

func ReachedThrough(vm, mac string) string {
	return "ssh " + vm + " goes through " + mac + ": station machines"
}

func RunOnItself(machine string) string { return "run it on " + machine + " itself: ssh " + machine }

const MachinesSSHUse = "ssh <machine>"

func SSHConfigNotRefreshed(cause string) string { return "ssh config not refreshed: " + cause }
