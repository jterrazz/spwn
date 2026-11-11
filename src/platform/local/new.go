package local

import (
	"io"
	"path/filepath"
	"slices"
	"strings"
)

const defaultSystemDirs = "/usr/sbin:/sbin:/usr/libexec:/usr/libexec/ApplicationFirewall"

type Search struct {
	// value: the folders an install lands commands in, searched after PATH and
	// the system dirs, and appended to a child's PATH once they exist; their
	// owners declare them and wire hands them here, so the runner knows no
	// manager. System's are absolute folders of the system, read under
	// STATION_SYSTEM_ROOT.
	System []string
	// value: this user's own absolute folders, under HOME, which a fence
	// points into the run's own folder; taken as they are.
	Home []string
}

func New(env []string, stdin io.Reader, stdout, stderr io.Writer) System {
	return NewSearching(env, stdin, stdout, stderr, Search{})
}

func NewSearching(env []string, stdin io.Reader, stdout, stderr io.Writer, search Search) System {
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
	landing := search.under(vars)
	system := strings.Split(systemDirs, ":")
	return System{
		Run:     &runner{env: env, dirs: searched(vars["PATH"], system, landing), system: system, landing: landing, stdin: stdin, stdout: stdout, stderr: stderr},
		FS:      disk{},
		Env:     identity{vars: vars},
		Net:     network{},
		Sockets: sockets{},
		Clock:   clock{},
	}
}

func searched(path string, system, landing []string) []string {
	var dirs []string
	for _, d := range slices.Concat(strings.Split(path, ":"), system, landing) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// invariant: a program is looked up on disk at each call, so a manager or a
// script installed earlier in the run is found by the next step; the folders it
// lands in are listed up front. The system's are read under STATION_SYSTEM_ROOT
// as every other absolute path of the catalogue, so a fenced run (an empty root of
// its own) never meets the owner's, and with a fence set but no absolute root
// there are none.
func (s Search) under(vars map[string]string) []string {
	var dirs []string
	root := vars["STATION_SYSTEM_ROOT"]
	_, fenced := vars["STATION_SYSTEM_DIRS"]
	for _, d := range s.System {
		switch {
		case filepath.IsAbs(root):
			dirs = append(dirs, filepath.Join(root, d))
		case !fenced:
			dirs = append(dirs, d)
		}
	}
	for _, d := range s.Home {
		if filepath.IsAbs(d) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}
