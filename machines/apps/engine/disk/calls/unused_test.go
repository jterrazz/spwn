package calls

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

func unusedStation() *changingStation {
	return &changingStation{held: &disk.Held{Unused: []disk.Freeable{
		{Holder: disk.HolderImages, Kind: disk.FreeImage, Name: "base-ubuntu@2026.10.20", Bytes: 30},
		{Holder: disk.HolderLeftovers, Kind: disk.FreeDisk, Name: "gone", Bytes: 40},
	}}}
}

func freeUnused(s *Surveyor, in FreeUnusedIn) ([]disk.Outcome, error) {
	return FreeUnused.Run(context.Background(), Deps{Disk: s, Kind: machine.Computer, Name: "mbp"}, in, nil)
}

func TestFreeUnusedAsksTheNamesTypedThenFreesThem(t *testing.T) {
	station := unusedStation()
	both := []string{"base-ubuntu@2026.10.20", "gone", "gone"}
	_, err := freeUnused(onMac(station), FreeUnusedIn{Names: both, Confirm: confirm.Yes})
	var asked *fail.Error
	if !errors.As(err, &asked) || asked.Code != confirm.Needed || asked.Subject != "base-ubuntu@2026.10.20, gone" || len(station.freed) != 0 {
		t.Fatalf("answered yes: %v, freed %v; want the names asked typed, each once, and nothing freed", err, station.freed)
	}
	answer, ok := confirm.Answer(err)
	if !ok || answer != "base-ubuntu@2026.10.20, gone" {
		t.Fatalf("the asking side answers %q (%v), want the names it was shown", answer, ok)
	}
	out, err := freeUnused(onMac(station), FreeUnusedIn{Names: both, Confirm: answer})
	if err != nil || len(out) != 2 || !slices.Equal(station.freed, []string{"base-ubuntu@2026.10.20", "gone"}) {
		t.Fatalf("typed: %v %+v, freed %v; want both freed once", err, out, station.freed)
	}
}

func TestATypedAnswerFreesOnlyTheSetItNamed(t *testing.T) {
	station := unusedStation()
	_, err := freeUnused(onMac(station), FreeUnusedIn{Names: []string{"base-ubuntu@2026.10.20", "gone"}, Confirm: "gone"})
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != confirm.Mismatch || len(station.freed) != 0 {
		t.Fatalf("typed gone for both: %v, freed %v; want a mismatch and nothing freed", err, station.freed)
	}
}

func TestFreeUnusedNamesWhatItFrees(t *testing.T) {
	station := unusedStation()
	_, err := freeUnused(onMac(station), FreeUnusedIn{Confirm: confirm.Yes})
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != NamesNeeded || len(station.freed) != 0 {
		t.Fatalf("asked no name: %v, freed %v; want disk.names-needed and nothing freed", err, station.freed)
	}
}

func TestFreeUnusedRefusesANameNothingUsesNoLongerHolds(t *testing.T) {
	station := unusedStation()
	_, err := freeUnused(onMac(station), FreeUnusedIn{Names: []string{"gone", "linux-1"}, Confirm: "gone, linux-1"})
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != NotFreeable || refused.Subject != "linux-1" || len(station.freed) != 0 {
		t.Fatalf("asked linux-1, which a machine record names: %v, freed %v; want disk.not-freeable before anything goes", err, station.freed)
	}
	if _, err := freeUnused(onMac(station), FreeUnusedIn{Names: []string{"gone"}, Confirm: "gone"}); err != nil || !slices.Equal(station.freed, []string{"gone"}) {
		t.Fatalf("asked gone alone: %v, freed %v; want gone alone", err, station.freed)
	}
}

func TestAnUnreadRecordRefusesTheWholeFree(t *testing.T) {
	station := unusedStation()
	station.held.Unread = "~/.local/state/station/vms/linux-1.json"
	_, err := freeUnused(onMac(station), FreeUnusedIn{Names: []string{"gone"}, Confirm: "gone"})
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != RecordUnreadable || refused.Subject != station.held.Unread || len(station.freed) != 0 {
		t.Fatalf("a record unread: %v, freed %v; want disk.record-unreadable and nothing freed", err, station.freed)
	}
}

func TestEachCanFreeNamesTheCallThatFreesIt(t *testing.T) {
	if Free.Name != disk.FreeCall || FreeUnused.Name != disk.FreeUnusedCall {
		t.Fatalf("calls %q and %q, the readings name %q and %q", Free.Name, FreeUnused.Name, disk.FreeCall, disk.FreeUnusedCall)
	}
}
