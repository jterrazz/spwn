package store

import (
	"context"
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

type mac struct {
	VMs []string `json:"vms"`
}

var t0 = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

func fresh(t *testing.T) *Store {
	t.Helper()
	return New(EpochOf(t0, 4123), t0, func(error) *fail.Error { return nil })
}

func put(t *testing.T, s *Store, key Key, value any, at time.Time) {
	t.Helper()
	if err := s.Put(Put{Key: key, Value: value, ReadAt: at}); err != nil {
		t.Fatal(err)
	}
}

func wantRev(t *testing.T, s *Store, want uint64) {
	t.Helper()
	if got := s.Now().Rev; got != want {
		t.Fatalf("rev %d, want %d", got, want)
	}
}

const mini Key = "mac:mac-mini"

func TestTheSameValueReadTwiceMakesOneRevision(t *testing.T) {
	s := fresh(t)
	put(t, s, mini, mac{VMs: []string{"linux-1"}}, t0)
	put(t, s, mini, mac{VMs: []string{"linux-1"}}, t0.Add(30*time.Second))
	wantRev(t, s, 1)
	if got := s.Now().Slices[mini].ReadAt; !got.Equal(t0) {
		t.Fatalf("a published revision moved its read_at to %v", got)
	}
	put(t, s, mini, mac{VMs: []string{"linux-1", "linux-2"}}, t0.Add(60*time.Second))
	wantRev(t, s, 2)
	if sl := s.Now().Slices[mini]; !sl.ReadAt.Equal(t0.Add(60*time.Second)) || sl.Rev != 2 || string(sl.Raw) != `{"vms":["linux-1","linux-2"]}` || sl.Hash != "6c40be63ed83761c" {
		t.Fatalf("changed slice %+v", sl)
	}
}

func TestAStateChangeKeepsTheValueAndItsHash(t *testing.T) {
	s := fresh(t)
	put(t, s, mini, mac{VMs: []string{"linux-1"}}, t0)
	before := s.Now().Slices[mini]
	if err := s.Fail(mini, WhyUnreachable); err != nil {
		t.Fatal(err)
	}
	wantRev(t, s, 2)
	after := s.Now().Slices[mini]
	if after.State != Stale || after.Why != WhyUnreachable || after.Hash != before.Hash || string(after.Raw) != string(before.Raw) || !after.ReadAt.Equal(t0) {
		t.Fatalf("failed slice %+v, was %+v", after, before)
	}
	if err := s.Fail(mini, WhyUnreachable); err != nil {
		t.Fatal(err)
	}
	wantRev(t, s, 2)
	if err := s.Fail(mini, WhyLate); err != nil {
		t.Fatal(err)
	}
	wantRev(t, s, 3)
	put(t, s, mini, mac{VMs: []string{"linux-1"}}, t0.Add(time.Minute))
	wantRev(t, s, 4)
	if back := s.Now().Slices[mini]; back.State != Fresh || back.Why != "" || back.Hash != before.Hash {
		t.Fatalf("read again %+v", back)
	}
}

func TestAKeyNeverReadStaysUnread(t *testing.T) {
	s := fresh(t)
	if err := s.Fail(mini, WhyReading); err != nil {
		t.Fatal(err)
	}
	if err := s.Fail(mini, WhyFarTooOld); err != nil {
		t.Fatal(err)
	}
	wantRev(t, s, 2)
	if sl := s.Now().Slices[mini]; sl.State != Unread || sl.Why != WhyFarTooOld || sl.Value != nil || sl.Raw != nil {
		t.Fatalf("unread slice %+v", sl)
	}
}

func TestSweepTurnsASliceStalePastTwiceItsPeriodAndItsBudget(t *testing.T) {
	s := fresh(t)
	ports := Of(FamilyPorts, "mac-mini")
	awake := Of(FamilyAwake, "mac-mini")
	put(t, s, mini, mac{}, t0)
	put(t, s, ports, mac{}, t0)
	put(t, s, awake, mac{}, t0)
	put(t, s, mini, mac{}, t0.Add(10*time.Second))
	cadence := func(k Key) (Cadence, bool) {
		if k.Family() == FamilyAwake {
			return Cadence{}, false
		}
		return Cadence{Every: 30 * time.Second, Budget: 5 * time.Second}, true
	}
	s.Sweep(t0.Add(65*time.Second), cadence)
	wantRev(t, s, 3)
	s.Sweep(t0.Add(65*time.Second+time.Nanosecond), cadence)
	wantRev(t, s, 4)
	now := s.Now().Slices
	if now[ports].State != Stale || now[ports].Why != WhyLate {
		t.Fatalf("ports %+v", now[ports])
	}
	if now[mini].State != Fresh || now[awake].State != Fresh {
		t.Fatalf("swept a slice read since or read on change: %+v %+v", now[mini], now[awake])
	}
	s.Sweep(t0.Add(time.Hour), cadence)
	wantRev(t, s, 5)
	s.Sweep(t0.Add(2*time.Hour), cadence)
	wantRev(t, s, 5)
}

func TestTheClockAdvancesEveryFifteenSeconds(t *testing.T) {
	s := fresh(t)
	s.Clock(t0.Add(14 * time.Second))
	wantRev(t, s, 0)
	s.Clock(t0.Add(15 * time.Second))
	wantRev(t, s, 1)
	if at := s.Now().At; !at.Equal(t0.Add(15 * time.Second)) {
		t.Fatalf("at %v", at)
	}
	changed, gone, ok := s.Since(s.Now().Epoch, 0)
	if !ok || len(changed) != 0 || len(gone) != 0 {
		t.Fatalf("a clock revision changed %v gone %v ok %v", changed, gone, ok)
	}
}

func TestSinceAnswersTheUnionOfWhatMoved(t *testing.T) {
	s := fresh(t)
	ep := s.Now().Epoch
	a, b, c := Of(FamilyMac, "a"), Of(FamilyMac, "b"), Of(FamilyMac, "c")
	put(t, s, a, mac{}, t0)
	put(t, s, b, mac{}, t0)
	put(t, s, c, mac{}, t0)
	s.Drop(b)
	s.Drop(c)
	put(t, s, c, mac{VMs: []string{"x"}}, t0)
	s.Drop(Of(FamilyMac, "never"))
	wantRev(t, s, 6)
	changed, gone, ok := s.Since(ep, 1)
	if !ok || !reflect.DeepEqual(changed, []Key{c}) || !reflect.DeepEqual(gone, []Key{b}) {
		t.Fatalf("since 1: changed %v gone %v ok %v", changed, gone, ok)
	}
	if changed, gone, ok := s.Since(ep, 6); !ok || len(changed) != 0 || len(gone) != 0 {
		t.Fatalf("since now: changed %v gone %v ok %v", changed, gone, ok)
	}
	if _, _, ok := s.Since(ep, 7); ok {
		t.Fatal("a revision ahead of the store resumed")
	}
	if _, _, ok := s.Since(EpochOf(t0, 4124), 1); ok {
		t.Fatal("another engine's revision resumed")
	}
}

func TestSinceSaysSnapshotOnceTheRevisionLeftTheRing(t *testing.T) {
	s := fresh(t)
	ep := s.Now().Epoch
	for i := range kept + 1 {
		put(t, s, mini, mac{VMs: make([]string, i)}, t0)
	}
	if _, _, ok := s.Since(ep, 1); !ok {
		t.Fatal("the oldest revision of the ring did not resume")
	}
	if _, _, ok := s.Since(ep, 0); ok {
		t.Fatal("an evicted revision resumed")
	}
}

type op struct {
	put  Key
	fail Key
	drop Key
	vms  int
	why  string
}

func apply(t *testing.T, s *Store, o op, at time.Time) {
	t.Helper()
	switch {
	case o.put != "":
		put(t, s, o.put, mac{VMs: make([]string, o.vms)}, at)
	case o.fail != "":
		if err := s.Fail(o.fail, o.why); err != nil {
			t.Fatal(err)
		}
	case o.drop != "":
		s.Drop(o.drop)
	default:
		s.Clock(at)
	}
}

func TestAClientApplyingEveryChangeEqualsTheStore(t *testing.T) {
	a, b, v := Of(FamilyMac, "a"), Of(FamilyOverview, "linux-1"), Key(FamilyVault)
	ops := []op{
		{put: a, vms: 1}, {fail: b, why: WhyReading}, {put: v}, {put: b, vms: 2},
		{put: a, vms: 1}, {fail: a, why: WhyLate}, {drop: v}, {}, {put: v, vms: 3},
		{fail: a, why: WhyUnreachable}, {put: a, vms: 1}, {drop: b}, {fail: b, why: WhyFarTooOld},
		{put: b, vms: 4}, {drop: a}, {put: a, vms: 5}, {},
	}
	for _, every := range []int{1, 2, 3, 5, len(ops)} {
		s := fresh(t)
		client := s.Now()
		mirror := maps.Clone(client.Slices)
		for i, o := range ops {
			apply(t, s, o, t0.Add(time.Duration(i+1)*ClockEvery))
			if (i+1)%every != 0 && i != len(ops)-1 {
				continue
			}
			changed, gone, ok := s.Since(client.Epoch, client.Rev)
			if !ok {
				t.Fatalf("every %d, op %d: no resume from rev %d", every, i, client.Rev)
			}
			client = s.Now()
			for _, k := range changed {
				mirror[k] = client.Slices[k]
			}
			for _, k := range gone {
				delete(mirror, k)
			}
			if !reflect.DeepEqual(mirror, client.Slices) {
				t.Fatalf("every %d, op %d: client %+v\nstore %+v", every, i, mirror, client.Slices)
			}
		}
	}
}

func TestAWatcherIsHandedTheLatestSnapshotOnly(t *testing.T) {
	s := fresh(t)
	ctx, cancel := context.WithCancel(context.Background())
	w := s.Watch(ctx)
	if first := <-w; first.Rev != 0 {
		t.Fatalf("first snapshot rev %d", first.Rev)
	}
	put(t, s, mini, mac{}, t0)
	put(t, s, mini, mac{VMs: []string{"x"}}, t0)
	if got := <-w; got.Rev != 2 {
		t.Fatalf("handed rev %d, want the latest, 2", got.Rev)
	}
	cancel()
	if _, open := <-w; open {
		t.Fatal("a cancelled watch stayed open")
	}
	put(t, s, mini, mac{}, t0)
}

func TestAPutThatCannotBeHeldChangesNothing(t *testing.T) {
	s := fresh(t)
	if err := s.Put(Put{Key: mini, Value: mac{}}, Put{Key: "mac", Value: mac{}}); err == nil {
		t.Fatal("a key with no machine was held")
	}
	if err := s.Put(Put{Key: mini, Value: mac{}}, Put{Key: Of(FamilyPorts, "a"), Value: make(chan int)}); err == nil {
		t.Fatal("a value with no JSON form was held")
	}
	if err := s.Fail("tailnet:a", WhyLate); err == nil {
		t.Fatal("a failure of a malformed key was held")
	}
	wantRev(t, s, 0)
}

func TestKeysAndPatterns(t *testing.T) {
	for _, tc := range []struct {
		in string
		ok bool
	}{
		{"mac:macbook-pro", true}, {"mac:manageds-virtual-machine.local", true}, {"mac:.local", false}, {"overview:linux-1", true}, {"tailnet", true}, {"records", true}, {"vault", true},
		{"mac", false}, {"vault:a", false}, {"apps:linux-1", false}, {"mac:Linux-1", false}, {"mac:", false}, {"mac:a_b", false},
	} {
		if _, err := ParseKey(tc.in); (err == nil) != tc.ok {
			t.Errorf("ParseKey(%q) err %v", tc.in, err)
		}
	}
	if k := Of(FamilyActivity, "linux-1"); k.Family() != FamilyActivity || k.Machine() != "linux-1" {
		t.Errorf("Of %q: %q %q", k, k.Family(), k.Machine())
	}
	if k := Of(FamilyRecords, "ignored"); k != "records" || k.Machine() != "" {
		t.Errorf("Of records %q", k)
	}
	for _, in := range []string{"tailnet:*", "apps:*", "mac:A", "nope"} {
		if _, err := ParsePattern(in); err == nil {
			t.Errorf("ParsePattern(%q) took it", in)
		}
	}
	thisMac := func(m string) bool { return m == "macbook-pro" || m == "linux-1" || m == "" }
	for _, tc := range []struct {
		pattern string
		key     Key
		match   bool
	}{
		{"local", "mac:macbook-pro", true}, {"local", "signins:linux-1", true}, {"local", "mac:mac-mini", false},
		{"local", "tailnet", false}, {"local", "vault", false},
		{"ports:*", "ports:mac-mini", true}, {"ports:*", "mac:mac-mini", false},
		{"mac:mac-mini", "mac:mac-mini", true}, {"mac:mac-mini", "mac:macbook-pro", false}, {"records", "records", true},
	} {
		p, err := ParsePattern(tc.pattern)
		if err != nil {
			t.Fatal(err)
		}
		if got := p.Match(tc.key, thisMac); got != tc.match {
			t.Errorf("%q matches %q: %v", tc.pattern, tc.key, got)
		}
	}
}

func TestAnEpochNamesTheEngineStartAndItsPid(t *testing.T) {
	if got := EpochOf(t0.In(time.FixedZone("CET", 3600)), 4123); got != "2026-11-21T09:00:00Z/4123" {
		t.Fatalf("epoch %q", got)
	}
}
