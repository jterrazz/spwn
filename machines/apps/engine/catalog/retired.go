package catalog

import (
	"errors"
	"strconv"
	"strings"
)

const TailnetCLI = "tailscale"

func SSHCommand(user, mac string) (string, error) {
	// invariant: the command is run by a shell, so a user or a host that is
	// not one plain word is single-quoted, and the shell reads each as one;
	// one that ssh would read as an option, a leading "-", is refused.
	for _, w := range []string{user, mac} {
		if strings.HasPrefix(w, "-") {
			return "", errors.New("ssh would read " + strconv.Quote(w) + " as an option")
		}
	}
	if user != "" {
		return "ssh " + shellWord(user) + "@" + shellWord(mac), nil
	}
	return "ssh " + shellWord(mac), nil
}

func shellWord(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
