package wiki_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

// value: the faces the approved prototype derived on the client for the seven
// pages it embeds, dumped by running its own functions (rankOf, firstSentence,
// secIcon, secLabel, intoTable, wordsAt and fmtT) in a browser.
type prototypePage struct {
	Title     string `json:"title"`
	HeadFirst string `json:"headFirst"`
	Decision  *struct {
		Number, Title, Decides string
	} `json:"decision"`
	Sections []struct {
		Label string `json:"label"`
		Icon  string `json:"icon"`
	} `json:"sections"`
	Labels []string        `json:"labels"`
	Blocks int             `json:"blocks"`
	Items  []prototypeItem `json:"items"`
}

type prototypeItem struct {
	Key        string          `json:"key"`
	Kind       string          `json:"kind"`
	Ranks      []int           `json:"ranks"`
	First      string          `json:"first"`
	More       bool            `json:"more"`
	Rows       *int            `json:"rows"`
	Mirrors    string          `json:"mirrors"`
	MirroredBy *string         `json:"mirroredBy"`
	Mirrored   []bool          `json:"mirrored"`
	Kids       []prototypeItem `json:"kids"`
}

func collapsed(s string) string { return strings.Join(strings.Fields(s), " ") }

func runsText(rs []wiki.Run) string {
	var b strings.Builder
	for _, r := range rs {
		b.WriteString(r.Text)
	}
	return b.String()
}

func prototypeKey(it wiki.ServedItem) string {
	switch {
	case it.Block != nil:
		return "b:" + it.Block.ID
	case it.Group != nil:
		return "g:" + it.Group.ID
	}
	return "p:" + strings.ReplaceAll(strings.TrimPrefix(it.Prose.ID, "p"), ".", ":")
}

// invariant: the engine serves what the prototype drew, item by item, but
// for one difference: a figure opens no further than its card, the face the
// prototype draws it at past rank 2 too. The reading labels differ by design
// and are not compared here: the engine counts the words each level shows,
// a page's title and its findings included, where the prototype counted fewer.
func TestTheFacesServedAreThoseThePrototypeDerived(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "prototype", "faces.json"))
	if err != nil {
		t.Fatal(err)
	}
	var proto map[string]prototypePage
	if err := json.Unmarshal(body, &proto); err != nil {
		t.Fatal(err)
	}
	served := servedCorpus(t)
	if len(proto) != len(served) {
		t.Fatalf("the prototype embeds %d pages, the corpus serves %d", len(proto), len(served))
	}
	for id, want := range proto {
		s := served[id]
		got := map[string]wiki.ServedItem{}
		for _, it := range everyItem(s) {
			got[prototypeKey(it)] = it
		}
		if s.Blocks != want.Blocks {
			t.Errorf("%s: %d blocks; the prototype counts %d", id, s.Blocks, want.Blocks)
		}
		if len(s.Sections) != len(want.Sections) {
			t.Fatalf("%s: %d sections; the prototype reads %d", id, len(s.Sections), len(want.Sections))
		}
		for i, sec := range want.Sections {
			if s.Sections[i].Label != sec.Label || s.Sections[i].Icon != sec.Icon {
				t.Errorf("%s section %d: %q %s; the prototype draws %q %s", id, i, s.Sections[i].Label, s.Sections[i].Icon, sec.Label, sec.Icon)
			}
		}
		if want.Decision == nil && collapsed(runsText(s.Lede.First)) != collapsed(want.HeadFirst) {
			t.Errorf("%s lede: %q; the prototype cuts %q", id, runsText(s.Lede.First), want.HeadFirst)
		}
		if d := want.Decision; d != nil && (s.Page.Decision == nil || s.Page.Decision.Number != d.Number || s.Page.Decision.Title != d.Title || s.Page.Decision.Decides != d.Decides) {
			t.Errorf("%s decision: %+v; the prototype found %+v", id, s.Page.Decision, d)
		}
		var items []prototypeItem
		for _, it := range want.Items {
			items = append(items, it)
			items = append(items, it.Kids...)
		}
		if len(items) != len(got) {
			t.Errorf("%s: %d items; the prototype draws %d", id, len(got), len(items))
		}
		for _, w := range items {
			it, ok := got[w.Key]
			if !ok {
				t.Errorf("%s: %s not served", id, w.Key)
				continue
			}
			ranks := make([]wiki.Detail, 0, len(w.Ranks))
			for _, r := range w.Ranks {
				ranks = append(ranks, wiki.Details()[r])
				if it.Block != nil && it.Block.Figure != nil && r > rankOf(wiki.DetailCard) {
					ranks[len(ranks)-1] = wiki.DetailCard
				}
			}
			if !slices.Equal(it.Ranks, ranks) {
				t.Errorf("%s %s: ranks %v; the prototype derived %v", id, w.Key, it.Ranks, ranks)
			}
			if it.Prose != nil && (collapsed(runsText(it.Prose.First)) != collapsed(w.First) || it.Prose.More != w.More) {
				t.Errorf("%s %s: first %q more %v; the prototype cut %q more %v", id, w.Key, runsText(it.Prose.First), it.Prose.More, w.First, w.More)
			}
			if w.Rows != nil && (it.Block == nil || it.Block.Table == nil || it.Block.Table.Count != *w.Rows || it.Block.Table.Mirrors != w.Mirrors) {
				t.Errorf("%s %s: a table of %d rows mirroring %q in the prototype; served %+v", id, w.Key, *w.Rows, w.Mirrors, it.Block)
			}
			if it.Group != nil {
				by := ""
				if w.MirroredBy != nil {
					by = *w.MirroredBy
				}
				var from wiki.Detail
				if i := slices.Index(w.Mirrored, true); i >= 0 {
					from = wiki.Details()[i]
				}
				if it.Group.MirroredBy != by || it.Group.MirroredFrom != from {
					t.Errorf("%s %s: mirrored by %q from %q; the prototype %q from %q", id, w.Key, it.Group.MirroredBy, it.Group.MirroredFrom, by, from)
				}
			}
		}
	}
}
