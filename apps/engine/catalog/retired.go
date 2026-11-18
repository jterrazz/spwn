package catalog

const TailnetCLI = "tailscale"

func SSHCommand(user, mac string) string {
	if user != "" {
		return "ssh " + user + "@" + mac
	}
	return "ssh " + mac
}
