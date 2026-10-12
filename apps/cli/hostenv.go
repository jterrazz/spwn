package cli

import (
	"os"
	"runtime"
	"strings"
)

func dockerFriendlyPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/opt/homebrew/bin",
			"/opt/homebrew/sbin",
			"/usr/local/bin",
			"/usr/local/sbin",
			"/Applications/Docker.app/Contents/Resources/bin",
		}
	case "linux":
		return []string{
			"/usr/local/bin",
			"/usr/local/sbin",
			"/snap/bin",
		}
	default:
		return nil
	}
}

// EnsureDockerFriendlyPATH prepends the well-known Docker / Homebrew
// install directories to the current process's PATH, de-duplicated and
// only if not already present. Safe to call multiple times.
//
// Context: on macOS, GUI applications launched by Finder or launchd
// receive a sanitized PATH (typically "/usr/bin:/bin:/usr/sbin:/sbin")
// that does NOT include /opt/homebrew/bin or /usr/local/bin where
// Docker Desktop installs its CLI. The Tauri desktop bundle spawns the
// spwn binary as a subprocess, so the spwn binary inherits that
// minimal PATH - meaning every `exec.Command("docker", ...)` call site
// fails to resolve the binary. Calling this function at process start
// fixes every call site at once without touching them individually.
//
// Returns true if PATH was modified.
func EnsureDockerFriendlyPATH() bool {
	extras := dockerFriendlyPaths()
	if len(extras) == 0 {
		return false
	}

	current := os.Getenv("PATH")
	existing := make(map[string]struct{})
	for _, p := range strings.Split(current, string(os.PathListSeparator)) {
		if p != "" {
			existing[p] = struct{}{}
		}
	}

	var toPrepend []string
	for _, p := range extras {
		if _, ok := existing[p]; ok {
			continue
		}
		toPrepend = append(toPrepend, p)
		existing[p] = struct{}{}
	}
	if len(toPrepend) == 0 {
		return false
	}

	sep := string(os.PathListSeparator)
	next := strings.Join(toPrepend, sep)
	if current != "" {
		next = next + sep + current
	}
	_ = os.Setenv("PATH", next)
	return true
}
