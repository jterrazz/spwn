package local

import (
	"io"
	"path/filepath"
	"strings"
)

const defaultSystemDirs = "/usr/sbin:/sbin:/usr/libexec:/usr/libexec/ApplicationFirewall"

func New(env []string, stdin io.Reader, stdout, stderr io.Writer) System {
	vars := map[string]string{}
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			vars[k] = v
		}
	}
	systemDirs, set := vars["STATION_SYSTEM_DIRS"]
	if !set {
		systemDirs = defaultSystemDirs
	}
	var dirs []string
	for _, d := range strings.Split(vars["PATH"]+":"+systemDirs+":"+strings.Join(managerDirs(vars), ":"), ":") {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return System{
		Run:     &runner{env: env, dirs: dirs, managers: managerDirs(vars), stdin: stdin, stdout: stdout, stderr: stderr},
		FS:      disk{},
		Env:     identity{vars: vars},
		Net:     network{},
		Sockets: sockets{},
		Clock:   clock{},
	}
}

// invariant: a program is looked up on disk at each call, so a manager or a
// script installed earlier in the run is found by the next step; the folders it
// lands in are listed up front. Homebrew's two are read under STATION_SYSTEM_ROOT
// as every other absolute path of the catalogue, so a fenced run (an empty root of
// its own) never meets the owner's Homebrew, and with a fence set but no absolute
// root there are none; ~/.local/bin and mise's shims are under HOME, which a fence
// points into the run's own folder.
func managerDirs(vars map[string]string) []string {
	var dirs []string
	root := vars["STATION_SYSTEM_ROOT"]
	_, fenced := vars["STATION_SYSTEM_DIRS"]
	switch {
	case filepath.IsAbs(root):
		dirs = append(dirs, filepath.Join(root, "/opt/homebrew/bin"), filepath.Join(root, "/usr/local/bin"))
	case !fenced:
		dirs = append(dirs, "/opt/homebrew/bin", "/usr/local/bin")
	}
	if home := vars["HOME"]; filepath.IsAbs(home) {
		data := vars["MISE_DATA_DIR"]
		if !filepath.IsAbs(data) {
			data = filepath.Join(home, ".local", "share", "mise")
		}
		dirs = append(dirs, filepath.Join(home, ".local", "bin"), filepath.Join(data, "shims"))
	}
	return dirs
}
