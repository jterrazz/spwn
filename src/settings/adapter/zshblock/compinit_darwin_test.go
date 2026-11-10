package zshblock_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/settings/adapter/zshblock"
)

// invariant: a fresh Mac's interactive zsh ran no compinit, so the block
// runs it and the completion it carries is defined; the proof runs macOS's
// own /bin/zsh, which a Linux runner does not ship.
func TestAFreshMacsZshGetsTheCompletionFromTheBlock(t *testing.T) {
	block := zshblock.NewBlock(&memory{files: map[string][]byte{}}, catalog.DotfilesFS, nil, zshblock.BlockOptions{ThisMachine: catalog.Mac})
	script, err := block.Zsh(context.Background(), "_station() { }\ncompdef _station station")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if err = os.WriteFile(filepath.Join(home, ".zshrc"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	sys := local.New([]string{"HOME=" + home, "ZDOTDIR=" + home, "PATH=/usr/bin:/bin"}, nil, nil, nil)
	out, err := sys.Run.Output(context.Background(), local.Command("/bin/zsh", "-ic", "whence -w _station compdef"))
	if err != nil || out.ExitCode != 0 {
		t.Fatalf("zsh -ic: exit %d, %v", out.ExitCode, err)
	}
	if got, want := out.Stdout, "_station: function\ncompdef: function\n"; got != want {
		t.Fatalf("whence -w = %q, want %q", got, want)
	}
}
