package local

import (
	"context"
	"sync"
)

const walkers = 8

func Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	// invariant: ONE walk sizes every root: eight workers share the roots' stacks
	// of directories, taking from the roots in turn (eight workers in one tree
	// contend on its directories: 9.6 s against 6 s for 192 roots, 2026-11-02), so
	// small roots answer at once; past ctx no worker takes another directory and
	// every root not yet answered answers -1, even one whose walk ended while the
	// cancel was being observed, so no size is ever reported after the cancel.
	// value: the walk is getattrlistbulk(2) on darwin and readdir with fstatat
	// elsewhere, chosen by the build; an adapter may not hold the unsafe call.
	w := &walk{ctx: ctx, stacks: make([][]string, len(roots)), left: make([]int, len(roots)), total: make([]int64, len(roots)), devs: make([]uint64, len(roots))}
	w.cond = sync.NewCond(&w.mu)
	answered := make([]bool, len(roots))
	var say sync.Mutex
	answer := func(i int, bytes int64) {
		say.Lock()
		defer say.Unlock()
		if !answered[i] {
			answered[i] = true
			if ctx.Err() != nil {
				bytes = -1
			}
			each(i, bytes)
		}
	}
	for i, root := range roots {
		kind, bytes, dev := lstat(root)
		switch kind {
		case kindDir:
			w.stacks[i], w.left[i], w.devs[i] = []string{root}, 1, dev
			w.open++
		default:
			answer(i, bytes)
		}
	}
	stop := context.AfterFunc(ctx, func() {
		w.mu.Lock()
		w.stopped = true
		w.mu.Unlock()
		w.cond.Broadcast()
	})
	defer stop()
	var wg sync.WaitGroup
	for range walkers {
		wg.Go(func() {
			buf := make([]byte, 256<<10)
			for {
				i, dir, ok := w.take()
				if !ok {
					return
				}
				var found []string
				bytes := readDir(w.ctx, dir, w.devs[i], buf, func(sub string) { found = append(found, sub) })
				if total, done := w.finish(i, bytes, found); done {
					answer(i, total)
				}
			}
		})
	}
	wg.Wait()
	for i := range roots {
		answer(i, -1)
	}
}

type walk struct {
	ctx     context.Context
	mu      sync.Mutex
	cond    *sync.Cond
	stacks  [][]string
	left    []int
	total   []int64
	devs    []uint64
	open    int
	lowest  int
	next    int
	stopped bool
}

func (w *walk) take() (root int, dir string, ok bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for {
		if w.stopped || w.open == 0 {
			return 0, "", false
		}
		for k := range len(w.stacks) - w.lowest {
			i := w.lowest + (w.next-w.lowest+k+len(w.stacks))%(len(w.stacks)-w.lowest)
			if n := len(w.stacks[i]); n > 0 {
				dir := w.stacks[i][n-1]
				w.stacks[i] = w.stacks[i][:n-1]
				w.next = i + 1
				return i, dir, true
			}
		}
		w.cond.Wait()
	}
}

func (w *walk) finish(i int, bytes int64, found []string) (int64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.total[i] += bytes
	w.stacks[i] = append(w.stacks[i], found...)
	w.left[i] += len(found) - 1
	for w.lowest < len(w.stacks) && w.left[w.lowest] == 0 {
		w.lowest++
	}
	if len(found) > 0 {
		w.cond.Broadcast()
	}
	if w.left[i] > 0 || w.stopped || w.ctx.Err() != nil {
		return 0, false
	}
	w.open--
	if w.open == 0 {
		w.cond.Broadcast()
	}
	return w.total[i], true
}

type kind int

const (
	kindOther kind = iota
	kindDir
)

// value: a sparse VM disk reports its full virtual size; its blocks are what it
// costs, and a small file costs its length, so a file counts the smaller of the two.
func cost(length, blocks int64) int64 {
	return min(length, blocks)
}
