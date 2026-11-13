package fake_test

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
)

func TestJournal(t *testing.T) {
	contracttest.Journal(t, func(*testing.T) contracttest.JournalSubject {
		j := fake.NewJournal()
		j.World().Now = contracttest.Morning()
		return contracttest.JournalSubject{Journal: j, Now: &j.World().Now}
	})
}

func TestHooks(t *testing.T) {
	contracttest.Hooks(t, func(*testing.T) contracttest.HooksSubject {
		edit := agents.Event{Kind: agents.UseDone, Session: "s-1", Folder: "/Users/admin/app", Act: agents.ActEdit, Path: "/Users/admin/app/src/a.ts", Added: 3, Removed: 1, At: contracttest.Morning()}
		payload, _ := json.Marshal(edit)
		return contracttest.HooksSubject{
			Hooks:      fake.NewHooks("fake"),
			OwnersKept: func(*testing.T) bool { return true },
			Recorded:   []contracttest.Recorded{{Name: "an edit done", Payload: payload, Want: edit}},
			Unknown:    [][]byte{[]byte(`{"kind":"compacted","session":"s-1"}`), []byte(`{"kind":"began"}`)},
		}
	})
}

func TestSession(t *testing.T) {
	contracttest.Session(t, func(*testing.T) contracttest.SessionSubject {
		s := fake.NewSession()
		typed := func() []contracttest.Typed {
			var out []contracttest.Typed
			for _, t := range s.Held() {
				for _, text := range t.Typed {
					out = append(out, contracttest.Typed{Terminal: t.Address, Text: text})
				}
			}
			return out
		}
		return contracttest.SessionSubject{Session: s, Opened: s.Opened, Typed: typed, Lines: func() []string { return nil }, Says: s.Says}
	})
}

func TestForge(t *testing.T) {
	contracttest.Forge(t, func(*testing.T) contracttest.ForgeSubject {
		f := fake.NewForge(42)
		return contracttest.ForgeSubject{Forge: f, Folder: "/Users/admin/app", Open: 42, Merged: f.Merged}
	})
}

func TestRepo(t *testing.T) {
	contracttest.Repo(t, func(*testing.T) contracttest.RepoSubject {
		want := agents.Change{Files: 3, Plus: 86, Minus: 12}
		return contracttest.RepoSubject{Repo: fake.NewRepo("/Users/admin/app", want), Folder: "/Users/admin/app", Unchanged: "/Users/admin/api", Base: "main", Want: want}
	})
}

func TestMultiplexer(t *testing.T) {
	contracttest.Multiplexer(t, func(*testing.T) contracttest.MultiplexerSubject {
		h := fake.NewMultiplexer()
		return contracttest.MultiplexerSubject{Multiplexer: h, Reached: h.Reached, Attached: h.Attached}
	})
}
