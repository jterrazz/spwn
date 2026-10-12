//go:build !windows

package architect

import (
	"context"
	"testing"
	"time"
)

func TestLockWorldImageSerialisesHolders(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	first, err := lockWorldImage(context.Background())
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}

	waiting, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := lockWorldImage(waiting); err == nil {
		t.Fatal("second lock was granted while the first was held")
	}

	first()
	first()
	second, err := lockWorldImage(context.Background())
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	second()
}
