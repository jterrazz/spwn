package local

import (
	"os"
	"os/user"
	"runtime"
)

type identity struct{ vars map[string]string }

func (i identity) Executable() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return path
}

func (i identity) Home() string {
	if home := i.vars["HOME"]; home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return home
}

func (i identity) User() string {
	if name := i.vars["USER"]; name != "" {
		return name
	}
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

func (i identity) Shell() string { return i.vars["SHELL"] }

func (i identity) Hostname() string {
	name, _ := os.Hostname()
	return name
}

func (i identity) Euid() int      { return os.Geteuid() }
func (i identity) GOOS() string   { return runtime.GOOS }
func (i identity) GOARCH() string { return runtime.GOARCH }
