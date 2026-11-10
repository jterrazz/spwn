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
		Run:     &runner{env: env, dirs: dirs, stdin: stdin, stdout: stdout, stderr: stderr},
		FS:      disk{},
		Env:     identity{vars: vars},
		Net:     network{},
		Sockets: sockets{},
		Clock:   clock{},
	}
}

// invariant: a program is looked up on disk at each call, so a manager installed
// earlier in the run is found by the next step; the folders it lands in are
// listed up front, under STATION_SYSTEM_ROOT as every other absolute path of the
// catalogue, so a fenced run (an empty root of its own) never meets the owner's
// own Homebrew. The owner's own folders (~/.local/bin, mise's shims) stay on the
// PATH the process was started with, where a fence can empty them.
func managerDirs(vars map[string]string) []string {
	root := ""
	if r := vars["STATION_SYSTEM_ROOT"]; filepath.IsAbs(r) {
		root = r
	}
	return []string{
		filepath.Join(root, "/opt/homebrew/bin"),
		filepath.Join(root, "/usr/local/bin"),
	}
}
