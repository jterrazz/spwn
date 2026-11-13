package jsonfile_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func files() local.FS { return local.New(nil, nil, nil, nil).FS }

type clock struct{ now *time.Time }

func (c clock) Now() time.Time { return *c.now }

func (clock) Sleep(ctx context.Context, d time.Duration) error {
	waited, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	<-waited.Done()
	return ctx.Err()
}

func at(now time.Time) clock { return clock{now: &now} }

func TestJournal(t *testing.T) {
	contracttest.Journal(t, func(t *testing.T) contracttest.JournalSubject {
		now := contracttest.Morning()
		j := jsonfile.New(files(), filepath.Join(t.TempDir(), "agents"), clock{now: &now}, 10*time.Millisecond)
		return contracttest.JournalSubject{Journal: j, Now: &now}
	})
}

func TestASessionIsOneFileOnlyItsOwnerReads(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	j := jsonfile.New(files(), dir, at(contracttest.Morning()), time.Second)
	if err := j.Write(context.Background(), agents.Agent{Session: "s-1", State: agents.Working}); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "s-1.json" {
		t.Fatalf("the folder holds %v, want s-1.json alone", entries)
	}
	info, err := files().Stat(filepath.Join(dir, "s-1.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, %v; want 0600", info.Mode().Perm(), err)
	}
}

type renameFails struct{ jsonfile.Files }

func (renameFails) Rename(string, string) error { return errors.New("disk full") }

func TestAWriteThatFailsLeavesTheSessionAsItWas(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	if err := jsonfile.New(files(), dir, at(contracttest.Morning()), time.Second).Write(context.Background(), agents.Agent{Session: "s-1", State: agents.Working}); err != nil {
		t.Fatal(err)
	}
	broken := jsonfile.New(renameFails{files()}, dir, at(contracttest.Morning()), time.Second)
	if err := broken.Write(context.Background(), agents.Agent{Session: "s-1", State: agents.Waiting}); err == nil {
		t.Fatal("a write whose rename failed succeeded")
	}
	got, err := broken.Read(context.Background())
	if err != nil || len(got) != 1 || got[0].State != agents.Working {
		t.Errorf("Read = %+v, %v; want s-1 still working", got, err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("the failed write left %v", entries)
	}
}

func TestASessionPastItsKeepIsDeletedFromTheDisk(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	now := contracttest.Morning()
	j := jsonfile.New(files(), dir, clock{now: &now}, time.Second)
	if err := j.Write(context.Background(), agents.Agent{Session: "s-old", State: agents.Finished, Ended: now}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(catalog.AgentKeptFor + time.Minute)
	if err := j.Write(context.Background(), agents.Agent{Session: "s-new", State: agents.Working}); err != nil {
		t.Fatal(err)
	}
	if _, err := files().Stat(filepath.Join(dir, "s-old.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("s-old.json is still on the disk: %v", err)
	}
}

func TestAJournalHoldsNoCommandAndNoPrompt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agents")
	j := jsonfile.New(files(), dir, at(contracttest.Morning()), time.Second)
	a := agents.Judge(agents.Agent{},
		agents.Event{Kind: agents.Began, Session: "s-1", Folder: "/Users/admin/app", At: contracttest.Morning()},
		agents.Event{Kind: agents.Prompted, Prompt: "deploy with token sk-ant-api03-" + strings.Repeat("k", 40) + "\nand never tell", At: contracttest.Morning()},
		agents.Event{Kind: agents.UseStarting, Act: agents.ActRun, Command: "curl -H 'Authorization: hunter2' https://example.com", At: contracttest.Morning()},
		agents.Event{Kind: agents.UseDone, Act: agents.ActRun, Output: "secret output line", At: contracttest.Morning()},
	)
	if err := j.Write(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "s-1.json"))
	for _, never := range []string{"sk-ant", "hunter2", "curl", "secret output", "never tell"} {
		if strings.Contains(string(data), never) {
			t.Errorf("the journal keeps %q:\n%s", never, data)
		}
	}
}
