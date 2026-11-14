package fake

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Shares struct {
	// invariant: the shares one Mac of the world keeps, by VM; a VM it has no
	// record of keeps none.
	w   *world.World
	mac string
}

var (
	_ contract.ShareStore = (*Shares)(nil)
	_ store.Migrator      = (*Shares)(nil)
)

func SharesOf(w *world.World, mac string) *Shares { return &Shares{w: w, mac: mac} }

func NewShares(vms ...string) *Shares {
	w := &world.World{ThisMac: "this-mac"}
	for _, vm := range vms {
		w.Mac("this-mac").VMs[vm] = &world.VM{Kept: &world.Kept{}}
	}
	return SharesOf(w, "this-mac")
}

func (s *Shares) kept() map[string][]accounts.Share {
	st := s.w.Mac(s.mac).Store()
	if st.Shares == nil {
		st.Shares = map[string][]accounts.Share{}
	}
	return st.Shares
}

func (s *Shares) Get(_ context.Context, vm string) ([]accounts.Share, error) {
	s.w.Lock()
	defer s.w.Unlock()
	return slices.Clone(s.kept()[vm]), nil
}

func (s *Shares) Put(_ context.Context, vm string, shares []accounts.Share) error {
	s.w.Lock()
	defer s.w.Unlock()
	if v, ok := s.w.Mac(s.mac).VMs[vm]; !ok || v.Kept == nil {
		return errors.Join(store.ErrNotFound, errors.New("no VM "+vm))
	}
	if len(shares) == 0 {
		delete(s.kept(), vm)
		return nil
	}
	s.kept()[vm] = slices.Clone(shares)
	return nil
}

func (s *Shares) List(context.Context) (map[string][]accounts.Share, error) {
	s.w.Lock()
	defer s.w.Unlock()
	return maps.Clone(s.kept()), nil
}

func (s *Shares) Delete(_ context.Context, vm string) error {
	s.w.Lock()
	defer s.w.Unlock()
	delete(s.kept(), vm)
	return nil
}

func (s *Shares) Migrate(context.Context) error {
	// invariant: as the file store moves the shares a v2 record kept: each
	// name a VM's record lends moves into the store once, shared at the
	// world's now for the default term, and leaves the record.
	s.w.Lock()
	defer s.w.Unlock()
	for _, vm := range slices.Sorted(maps.Keys(s.w.Mac(s.mac).VMs)) {
		v := s.w.Mac(s.mac).VMs[vm]
		if v.Kept == nil || len(v.Kept.Shared) == 0 {
			continue
		}
		for _, name := range v.Kept.Shared {
			if !slices.ContainsFunc(s.kept()[vm], func(have accounts.Share) bool { return have.Name == name }) {
				s.kept()[vm] = append(s.kept()[vm], accounts.Share{Name: name, At: s.w.Now, Until: s.w.Now.Add(accounts.DefaultShareTerm)})
			}
		}
		v.Kept.Shared = nil
	}
	return nil
}
