package words

func SettingsAsRoot(verb, pkg string) string { return "sudo station settings " + verb + " " + pkg }

func PasswordlessRestart(mac string) string {
	return "check that sudo needs no password for fdesetup on " + mac
}

func ReachedThrough(vm, mac string) string {
	return "ssh " + vm + " goes through " + mac + ": station machines"
}

func RunOnItself(machine string) string { return "run it on " + machine + " itself: ssh " + machine }

const MachinesSSHUse = "ssh <machine>"

func SSHConfigNotRefreshed(cause string) string { return "ssh config not refreshed: " + cause }
