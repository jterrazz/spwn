package process

import (
	"context"
	"sort"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

const (
	retryEvery  = time.Minute
	retryRetell = time.Hour
)

type GrantsOwed interface {
	// value: the grant removals this Mac still owes its VMs, by VM; Retry
	// takes them out again, Forget drops those of a VM that is gone.
	Owed() (map[string][]accounts.Program, error)
	Retry(ctx context.Context, vm string, programs []accounts.Program) error
	Forget(ctx context.Context, vm string) error
}

// invariant: the retry job: the removals owed are read at the start and
// every retryEvery; one owed to a running VM is taken out, with no
// notification. A removal that fails stays owed and is told once per
// retryRetell.
func (d *Engine) owedDue() bool {
	now := d.h.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.owedAt.IsZero() && now.Sub(d.owedAt) < retryEvery {
		return false
	}
	d.owedAt = now
	return true
}

func (d *Engine) sweepOwed(ctx context.Context) {
	if d.o.GrantsOwed == nil || !d.owedDue() {
		return
	}
	go func() {
		if !d.owedMu.TryLock() {
			return
		}
		defer d.owedMu.Unlock()
		d.retryOwed(ctx, "")
	}()
}

// invariant: a VM stopped while a removal was owed to it has it taken out
// at its next start, before its guest is reported connected.
func (d *Engine) retryBeforeReady(ctx context.Context, vm string) {
	if d.o.GrantsOwed == nil {
		return
	}
	d.owedMu.Lock()
	defer d.owedMu.Unlock()
	d.retryOwed(ctx, vm)
}

func (d *Engine) retryOwed(ctx context.Context, only string) {
	owed, err := d.o.GrantsOwed.Owed()
	if err != nil {
		return
	}
	vms := make([]string, 0, len(owed))
	for vm := range owed {
		vms = append(vms, vm)
	}
	sort.Strings(vms)
	for _, vm := range vms {
		if only != "" && vm != only || only == "" && !d.isConnected(vm) {
			continue
		}
		if len(owed[vm]) > 0 {
			d.retryOn(ctx, vm, owed[vm])
		}
	}
}

// invariant: the sweep retries only through a connected guest; a VM still
// connecting has its removals retried as it connects, before it is ready.
func (d *Engine) isConnected(vm string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	v := d.vms[vm]
	return v != nil && v.state == engine.StateRunning && v.guest == engine.GuestConnected
}

func (d *Engine) retryOn(ctx context.Context, vm string, programs []accounts.Program) {
	err := d.o.GrantsOwed.Retry(ctx, vm, programs)
	now := d.h.Now()
	for _, p := range programs {
		key := vm + " " + string(p)
		d.mu.Lock()
		if err == nil {
			delete(d.retryFailed, key)
			d.mu.Unlock()
			continue
		}
		last, told := d.retryFailed[key]
		retell := !told || now.Sub(last) >= retryRetell
		if retell {
			d.retryFailed[key] = now
		}
		d.mu.Unlock()
		if retell {
			d.notify(ctx, engine.Notification{Kind: engine.NoticeGrantRemovalFailed, Subject: vm + " grant", Text: "removing the grant of " + p.Label() + " failed: " + err.Error()})
		}
	}
}

// invariant: a VM deleted takes what was granted to it with its disk; the
// removals owed to it go with it.
func (d *Engine) forgetGrants(ctx context.Context, vm string) {
	if d.o.GrantsOwed != nil {
		_ = d.o.GrantsOwed.Forget(ctx, vm)
	}
}
