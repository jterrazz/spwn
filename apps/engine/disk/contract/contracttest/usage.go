package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/disk/contract"
)

func Scene() map[string]int {
	// value: the files every adapter's scene holds, by their path under its
	// root and their length.
	return map[string]int{"cache/a": 100, "cache/deep/b": 5000, "data/c": 300}
}

type Open func(t *testing.T, files map[string]int) (contract.DiskUsage, string)

func size(ctx context.Context, u contract.DiskUsage, root string) int64 {
	var got int64
	u.Sizes(ctx, []string{root}, func(_ int, b int64) { got = b })
	return got
}

func DiskUsage(t *testing.T, open Open) {
	t.Run("a folder is sized by the files under it", func(t *testing.T) {
		u, root := open(t, Scene())
		if got := size(context.Background(), u, root+"/cache"); got != 5100 {
			t.Fatalf("cache = %d, want 5100", got)
		}
		if got := size(context.Background(), u, root+"/missing"); got != 0 {
			t.Fatalf("a missing folder = %d, want 0", got)
		}
	})
	t.Run("a size asked past its context is unknown", func(t *testing.T) {
		u, root := open(t, Scene())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got := size(ctx, u, root+"/cache"); got != -1 {
			t.Fatalf("a cancelled size = %d, want -1", got)
		}
	})
	t.Run("an erased folder is gone, and the others are kept", func(t *testing.T) {
		u, root := open(t, Scene())
		if err := u.Erase(context.Background(), []string{root + "/cache"}); err != nil {
			t.Fatalf("Erase: %v", err)
		}
		if got := size(context.Background(), u, root+"/cache"); got != 0 {
			t.Fatalf("an erased folder = %d, want 0", got)
		}
		if got := size(context.Background(), u, root+"/data"); got != 300 {
			t.Fatalf("a kept folder = %d, want 300", got)
		}
	})
	t.Run("a moved folder is whole at its new location and gone from the old", func(t *testing.T) {
		u, root := open(t, Scene())
		if err := u.Move(context.Background(), root+"/cache", root+"/moved/cache"); err != nil {
			t.Fatalf("Move: %v", err)
		}
		if got := size(context.Background(), u, root+"/moved/cache"); got != 5100 {
			t.Fatalf("the moved folder = %d, want 5100", got)
		}
		if got := size(context.Background(), u, root+"/cache"); got != 0 {
			t.Fatalf("the old location = %d, want 0", got)
		}
	})
	t.Run("a move onto a folder that holds anything is refused, and nothing moves", func(t *testing.T) {
		u, root := open(t, Scene())
		if err := u.Move(context.Background(), root+"/cache", root+"/data"); err == nil {
			t.Fatal("a move onto a full folder was let through")
		}
		if got := size(context.Background(), u, root+"/cache"); got != 5100 {
			t.Fatalf("the refused folder = %d, want 5100", got)
		}
	})
	t.Run("the volume has room within its size", func(t *testing.T) {
		u, _ := open(t, Scene())
		total, free, err := u.Volume(context.Background())
		if err != nil || total <= 0 || free < 0 || free > total {
			t.Fatalf("Volume = %d total, %d free, %v", total, free, err)
		}
	})
	t.Run("an app that is not running is said so", func(t *testing.T) {
		u, _ := open(t, Scene())
		if u.Running(context.Background(), "station-no-such-app") {
			t.Fatal("a process that does not run was said running")
		}
	})
}
