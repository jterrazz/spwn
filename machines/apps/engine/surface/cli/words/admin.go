package words

func AdminApply(setting, mac string, thisMac bool) string {
	// value: the command that applies an admin setting with admin rights, on
	// its Mac's own terminal: station never runs sudo itself, from any
	// surface.
	command := "sudo station settings apply " + setting
	if thisMac {
		return command
	}
	return "ssh -t " + mac + " " + command
}
