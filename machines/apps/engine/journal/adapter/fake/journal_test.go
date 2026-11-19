package fake_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/journal/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestJournalStore(t *testing.T) {
	contracttest.Store(t, func(*testing.T) contract.Journal { return fake.New() })
}

func TestAJournalFileThatDoesNotReadIsRefused(t *testing.T) {
	w := &world.World{Macs: map[string]*world.Mac{"macbook-pro": {Files: map[string]string{fake.File: "{\"schema\":\"station.journal/v1\"}\nnot a line\n"}}}}
	var unread *store.UnreadableError
	if _, err := fake.Of(w, "macbook-pro").Lines(context.Background(), time.Time{}, time.Time{}); !errors.As(err, &unread) || unread.Path != fake.Path || unread.Line != 2 {
		t.Errorf("Lines = %v; want the journal unreadable at line 2", err)
	}
}
