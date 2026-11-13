package process

import (
	"context"
	"sort"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

const (
	expireEvery  = time.Minute
	expireRetell = time.Hour
)

type Shares interface {
	// value: the account shares of this Mac, by VM; Expire deletes a VM's
	// copies and their shares, Forget the shares of a VM that is gone.
	Held() (map[string][]accounts.Share, error)
	Expire(ctx context.Context, vm string, names []string) error
	Forget(ctx context.Context, vm string) error
}

// invariant: the expire job: the shares are read at the start and every
// expireEvery; a share past its until on a running VM is taken back, with no
// notification. A take-back that fails keeps its share and is told once per
// expireRetell.
func (d *Engine) sharesDue() bool {
	now := d.h.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.sharesAt.IsZero() && now.Sub(d.sharesAt) < expireEvery {
		return false
	}
	d.sharesAt = now
	return true
}

func (d *Engine) sweepShares(ctx context.Context) {
	if d.o.AccountShares == nil || !d.sharesDue() {
		return
	}
	go func() {
		if !d.shareMu.TryLock() {
			return
		}
		defer d.shareMu.Unlock()
		d.takeBackExpired(ctx, "")
	}()
}

// invariant: a VM stopped past a share's until has it taken back at its
// next start, before its guest is reported connected.
func (d *Engine) takeBackBeforeReady(ctx context.Context, vm string) {
	if d.o.AccountShares == nil {
		return
	}
	d.shareMu.Lock()
	defer d.shareMu.Unlock()
	d.takeBackExpired(ctx, vm)
}

func (d *Engine) takeBackExpired(ctx context.Context, only string) {
	held, err := d.o.AccountShares.Held()
	if err != nil {
		return
	}
	now := d.h.Now()
	vms := make([]string, 0, len(held))
	for vm := range held {
		vms = append(vms, vm)
	}
	sort.Strings(vms)
	for _, vm := range vms {
		if only != "" && vm != only || only == "" && !d.isConnected(vm) {
			continue
		}
		var names []string
		for _, s := range accounts.Expired(held[vm], now) {
			names = append(names, s.Name)
		}
		if len(names) > 0 {
			d.takeBack(ctx, vm, names)
		}
	}
}

// invariant: the sweep takes back only through a connected guest; a VM still
// connecting has its ended shares taken back as it connects, before it is ready.
func (d *Engine) isConnected(vm string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	v := d.vms[vm]
	return v != nil && v.state == engine.StateRunning && v.guest == engine.GuestConnected
}

func (d *Engine) takeBack(ctx context.Context, vm string, names []string) {
	err := d.o.AccountShares.Expire(ctx, vm, names)
	now := d.h.Now()
	for _, n := range names {
		key := vm + " " + n
		d.mu.Lock()
		if err == nil {
			delete(d.takeBackFailed, key)
			d.mu.Unlock()
			continue
		}
		last, told := d.takeBackFailed[key]
		retell := !told || now.Sub(last) >= expireRetell
		if retell {
			d.takeBackFailed[key] = now
		}
		d.mu.Unlock()
		if retell {
			d.notify(ctx, engine.Notification{Kind: engine.NoticeTakeBackFailed, Subject: vm + " share", Text: "taking " + accounts.Label(accounts.AccountOf(n)) + " back failed: " + err.Error()})
		}
	}
}

// invariant: a VM deleted takes its copies with its disk; its shares go
// with it.
func (d *Engine) forgetShares(ctx context.Context, vm string) {
	if d.o.AccountShares != nil {
		_ = d.o.AccountShares.Forget(ctx, vm)
	}
}
