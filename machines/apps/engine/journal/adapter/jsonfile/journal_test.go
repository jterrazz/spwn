package jsonfile_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/journal/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type lockable struct{ local.FS }

func (lockable) Lock(path string) (func(), error) { return local.Exclusive(context.Background(), path) }

func disk() jsonfile.Files { return lockable{local.New(nil, nil, nil, nil).FS} }

func TestJournalStore(t *testing.T) {
	contracttest.Store(t, func(t *testing.T) contract.Journal {
		return jsonfile.New(disk(), filepath.Join(t.TempDir(), "journal.jsonl"))
	})
}

func TestTheJournalIsVersionedOneLinePerLine(t *testing.T) {
	file := filepath.Join(t.TempDir(), "journal.jsonl")
	at := time.Date(2026, 11, 14, 14, 7, 0, 0, time.UTC)
	if err := jsonfile.New(disk(), file).Write(context.Background(), at, journal.Line{At: at, Machine: "gate-2", Text: "gate-2 deleted", By: "you", Outcome: journal.Done}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(file)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 || lines[0] != `{"schema":"station.journal/v1"}` || !strings.Contains(lines[1], `"Text":"gate-2 deleted"`) || strings.Contains(lines[1], `"Night"`) {
		t.Errorf("journal.jsonl =\n%s", data)
	}
}

func TestAJournalThatDoesNotReadIsRefusedAndANewerOneToo(t *testing.T) {
	file := filepath.Join(t.TempDir(), "journal.jsonl")
	s := jsonfile.New(disk(), file)
	_ = os.WriteFile(file, []byte("{\"schema\":\"station.journal/v1\"}\n{\"Text\":\n"), 0o600)
	var unread *store.UnreadableError
	if _, err := s.Lines(context.Background(), time.Time{}, time.Time{}); !errors.As(err, &unread) || unread.Line != 2 {
		t.Errorf("Lines of a broken journal = %v; want it unreadable at line 2", err)
	}
	_ = os.WriteFile(file, []byte("{\"schema\":\"station.journal/v2\"}\n"), 0o600)
	var newer *store.NewerError
	if _, err := s.Lines(context.Background(), time.Time{}, time.Time{}); !errors.As(err, &newer) {
		t.Errorf("Lines of a journal a newer station wrote = %v; want it refused", err)
	}
}
