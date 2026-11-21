package local

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func BundleOf(executable, rel string) string {
	// value: the app bundle that holds executable as rel, its links resolved
	// first, so a run through ~/.local/bin finds the bundle too; "" when the
	// file is not inside one.
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return ""
	}
	bundle, ok := strings.CutSuffix(resolved, string(filepath.Separator)+rel)
	if !ok || !strings.HasSuffix(bundle, ".app") {
		return ""
	}
	return bundle
}

type JobStart struct {
	// value: what a job of Spwn.app's engine is started with, which its static
	// definition inside the sealed bundle cannot hold: the environment the
	// installing shell had, and each job's log, by label.
	Env  map[string]string `json:"env"`
	Logs map[string]string `json:"logs"`
}

func ReadJobStart(path string) (JobStart, error) {
	var start JobStart
	body, err := os.ReadFile(path)
	if err != nil {
		return start, err
	}
	err = json.Unmarshal(body, &start)
	return start, err
}

func LogTo(path string) error {
	// invariant: the process's standard output and error both append to path
	// from here on, a crash's trace included, as launchd's StandardOutPath did.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	for _, fd := range []int{1, 2} {
		if err := unix.Dup2(int(f.Fd()), fd); err != nil {
			_ = f.Close()
			return err
		}
	}
	return f.Close()
}
