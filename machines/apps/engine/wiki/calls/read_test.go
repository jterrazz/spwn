package calls_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/wiki"
	"github.com/jterrazz/studio-station/apps/engine/wiki/calls"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

type counted struct {
	// value: a corpus that counts its whole reads, so a test sees whether a
	// read compiled anything.
	mu      sync.Mutex
	files   map[string]string
	reads   int
	refusal error
}

func (c *counted) Signature(context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refusal != nil {
		return "", c.refusal
	}
	var sig strings.Builder
	for _, p := range slices.Sorted(maps.Keys(c.files)) {
		sig.WriteString(p + "=" + c.files[p] + ";")
	}
	return sig.String(), nil
}

func localMac() machine.Profile { return machine.Profile{Caps: machine.Capabilities{Engine: true}} }

func (c *counted) Files(context.Context) (map[string]wiki.File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	out := map[string]wiki.File{}
	for p, body := range c.files {
		out[p] = wiki.File{Size: int64(len(body)), Body: []byte(body)}
	}
	return out, nil
}

const yaml = "wiki: spwn.wiki/v1\nfloors:\n  - {id: intention, name: Intention, question: Why}\n  - {id: product, name: Product, question: What}\n  - {id: domains, name: Domains, question: Which words}\n  - {id: architecture, name: Architecture, question: Where}\n  - {id: code, name: Code, question: Which lines}\nseats:\n  - {id: vision, name: Vision, path: [blueprint/], reads: [tell, define], home: blueprint/a}\nbudget: 2min\n"

func read(t *testing.T, h call.Handler, page string) error {
	t.Helper()
	_, err := h.Invoke(context.Background(), localMac(), []byte(`{"Page":"`+page+`"}`), nil)
	return err
}

func TestAnUnchangedCorpusIsServedFromMemoryAndAChangedOneIsReadAnew(t *testing.T) {
	c := &counted{files: map[string]string{"wiki.yaml": yaml, "blueprint/a.md": "# A\n", "blueprint/b.md": "# B\n"}}
	h := calls.All(calls.Deps{Corpus: c})[0]
	for range 3 {
		if err := read(t, h, "blueprint/a"); err != nil {
			t.Fatal(err)
		}
	}
	if c.reads != 1 {
		t.Fatalf("three reads of one corpus read its files %d times; want once", c.reads)
	}
	c.mu.Lock()
	c.files["blueprint/b.md"] = "# B, touched\n"
	c.mu.Unlock()
	if err := read(t, h, "blueprint/a"); err != nil || c.reads != 2 {
		t.Fatalf("a read after a page changed: %v, %d whole reads; want the corpus read anew", err, c.reads)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { _ = read(t, h, "blueprint/b") })
	}
	wg.Wait()
}

func TestEachWayACorpusFailsToReadIsCoded(t *testing.T) {
	for _, c := range []struct {
		refusal error
		code    fail.Code
	}{
		{contract.ErrNone, calls.None},
		{&wiki.TooLarge{Pages: 2001}, calls.TooLarge},
		{&contract.Unreadable{Path: "blueprint/secret.md", Err: errors.New("openat blueprint/secret.md: path escapes from parent")}, calls.Unreadable},
	} {
		h := calls.All(calls.Deps{Corpus: &counted{refusal: c.refusal}})[0]
		var e *fail.Error
		if err := read(t, h, ""); !errors.As(err, &e) || e.Code != c.code {
			t.Errorf("%v reads as %v; want %s", c.refusal, err, c.code)
		}
		if c.code == calls.Unreadable && (e.Subject != "blueprint/secret.md" || e.Detail["path"] != "blueprint/secret.md") {
			t.Errorf("the refusal names %q; want the path inside the wiki", e.Subject)
		}
	}
}
