package local

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const developerBudget = 5 * time.Second

// invariant: a shim of the shims folder is a program only while the developer
// directory `xcode-select -p` prints holds the program it stands for; else it
// is skipped as if absent, so no read of this machine opens macOS's install
// dialog. `xcode-select -p` only prints, it never asks; the install is the
// owner's to run. Nothing is cached, so an install made while the engine runs
// is found by its next read.
func (r *runner) shimmed(path string) bool {
	name := filepath.Base(path)
	if r.shims == "" || filepath.Dir(path) != r.shims || !r.shimNames[name] {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), developerBudget)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(r.shims, "xcode-select"), "-p")
	cmd.Env = r.env
	out, err := cmd.Output()
	dir := strings.TrimSpace(string(out))
	if err != nil || !filepath.IsAbs(dir) {
		return true
	}
	return !executable(filepath.Join(r.root, dir, "usr", "bin", name))
}
