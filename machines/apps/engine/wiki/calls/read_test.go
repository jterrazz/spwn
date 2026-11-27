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
	// value: when set, a whole read says it began on entered, then waits
	// for gate to close or for its ask to be abandoned; each signature read
	// is said on asked.
	gate    chan struct{}
	entered chan struct{}
	asked   chan struct{}
}

func gated() *counted {
	return &counted{files: map[string]string{"wiki.yaml": yaml, "blueprint/a.md": "# A\n"}, gate: make(chan struct{}), entered: make(chan struct{}, 16), asked: make(chan struct{}, 16)}
}

// value: waits until n asks have read the corpus's signature, the step
// before each joins the compile in progress or finds its corpus kept.
func (c *counted) askedBy(n int) {
	for range n {
		<-c.asked
	}
}

func TestEightAsksOfAColdCorpusCompileItOnce(t *testing.T) {
	c := gated()
	h := calls.All(calls.Deps{Corpus: c})[0]
	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := range 8 {
		wg.Go(func() { errs[i] = read(t, h, "blueprint/a") })
	}
	c.askedBy(8)
	close(c.gate)
	wg.Wait()
	if err := errors.Join(errs...); err != nil || c.reads != 1 {
		t.Fatalf("eight asks at once: %v, %d whole reads; want one read they all share", err, c.reads)
	}
}

func TestAnAskAbandonedWhileOthersWaitLeavesThemTheirCorpus(t *testing.T) {
	c := gated()
	h := calls.All(calls.Deps{Corpus: c})[0]
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() {
		_, err := h.Invoke(ctx, localMac(), []byte(`{"Page":"blueprint/a"}`), nil)
		first <- err
	}()
	<-c.entered
	second := make(chan error, 1)
	go func() { second <- read(t, h, "blueprint/a") }()
	c.askedBy(2)
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("the abandoned ask ends with %v; want context.Canceled", err)
	}
	close(c.gate)
	if err := <-second; err != nil || c.reads != 2 {
		t.Fatalf("the ask still waiting: %v, %d whole reads; want its corpus, read again", err, c.reads)
	}
}

func (c *counted) Signature(context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.asked != nil {
		c.asked <- struct{}{}
	}
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

func (c *counted) Files(ctx context.Context) (map[string]wiki.File, error) {
	c.mu.Lock()
	c.reads++
	gate := c.gate
	c.mu.Unlock()
	if gate != nil {
		c.entered <- struct{}{}
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
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

func TestTheMapAndTheReadOfAnUnchangedCorpusCompileItOnce(t *testing.T) {
	c := &counted{files: map[string]string{"wiki.yaml": yaml, "blueprint/a.md": "# A\n", "blueprint/b.md": "# B\n"}}
	handlers := calls.All(calls.Deps{Corpus: c})
	for range 2 {
		if err := read(t, handlers[0], "blueprint/a"); err != nil {
			t.Fatal(err)
		}
		if _, err := handlers[1].Invoke(context.Background(), localMac(), []byte(`{"Open":"blueprint/b"}`), nil); err != nil {
			t.Fatal(err)
		}
	}
	if c.reads != 1 {
		t.Fatalf("two reads and two maps of one corpus read its files %d times; want once", c.reads)
	}
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
		for _, h := range calls.All(calls.Deps{Corpus: &counted{refusal: c.refusal}}) {
			var e *fail.Error
			if _, err := h.Invoke(context.Background(), localMac(), []byte(`{}`), nil); !errors.As(err, &e) || e.Code != c.code {
				t.Errorf("%s: %v reads as %v; want %s", h.Meta().Name, c.refusal, err, c.code)
				continue
			}
			if c.code == calls.Unreadable && (e.Subject != "blueprint/secret.md" || e.Detail["path"] != "blueprint/secret.md") {
				t.Errorf("%s: the refusal names %q; want the path inside the wiki", h.Meta().Name, e.Subject)
			}
		}
	}
}
