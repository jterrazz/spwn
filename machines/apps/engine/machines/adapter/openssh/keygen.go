package openssh

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type KeyGen struct {
	run local.Runner
	fs  local.FS
}

func NewKeyGen(run local.Runner, fs local.FS) KeyGen { return KeyGen{run: run, fs: fs} }

func (k KeyGen) MakeKey(ctx context.Context, key string) error {
	// invariant: an image key is an ed25519 pair with no passphrase, named
	// for the image so the owner can tell it from their own keys.
	if err := k.fs.MkdirAll(path.Dir(key), 0o700); err != nil {
		return err
	}
	out, err := k.run.Output(ctx, local.Cmd{Name: "ssh-keygen", Args: []string{"-q", "-t", "ed25519", "-N", "", "-C", "station-image", "-f", key}})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return fmt.Errorf("ssh-keygen exited %d: %s", out.ExitCode, strings.TrimSpace(out.Stderr))
	}
	return nil
}
