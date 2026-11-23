package catalog

import "strings"

const TailnetCLI = "tailscale"

func SSHCommand(user, mac string) string {
	// invariant: the command is run by a shell, so a user or a host that is
	// not one plain word is single-quoted, and the shell reads each as one.
	if user != "" {
		return "ssh " + shellWord(user) + "@" + shellWord(mac)
	}
	return "ssh " + shellWord(mac)
}

func shellWord(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
