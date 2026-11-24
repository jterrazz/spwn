//go:build !windows

package compile

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLockTag_SerialisesHoldersOfOneTag(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	first, err := lockTag(context.Background(), "spwn/world:aaa")
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}

	waiting, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := lockTag(waiting, "spwn/world:aaa"); err == nil {
		t.Fatal("second lock of one tag was granted while the first was held")
	}

	first()
	first()
	second, err := lockTag(context.Background(), "spwn/world:aaa")
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	second()
}

func TestLockTag_LeavesOtherTagsFree(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	held, err := lockTag(context.Background(), "spwn/world:aaa")
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer held()

	waiting, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	other, err := lockTag(waiting, "spwn/world:bbb")
	if err != nil {
		t.Fatalf("a lock on another tag waited on the held one: %v", err)
	}
	other()
}

func TestLockTag_BoundsItsWait(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	bound := tagLockWait
	tagLockWait = 150 * time.Millisecond
	t.Cleanup(func() { tagLockWait = bound })

	held, err := lockTag(context.Background(), "spwn/world:aaa")
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer held()

	_, err = lockTag(context.Background(), "spwn/world:aaa")
	if err == nil || !strings.Contains(err.Error(), "another build of spwn/world:aaa held it past 150ms") {
		t.Fatalf("a wait past the bound gave %v", err)
	}
}
