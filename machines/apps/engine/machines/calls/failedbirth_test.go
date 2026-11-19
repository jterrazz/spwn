package calls_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

// value: a bench whose create of web failed at its boot, its console having
// written four lines by then.
func failedAtBoot(t *testing.T) bench {
	t.Helper()
	b := creating(t, true)
	b.engine.run = func() {
		b.w.Lock()
		defer b.w.Unlock()
		b.w.Mac("mac").VMs["web"].Console = []string{"boot: one", "boot: two", "boot: three", "boot: four"}
	}
	_, err := b.m.Create(context.Background(), model.CreateRequest{Image: "base-ubuntu", Name: "web", Lifetime: model.Persistent}, nil, nil)
	if !errors.As(err, new(*model.DidNotBootError)) {
		t.Fatalf("create: %v; want it not booted", err)
	}
	return b
}

func TestACreateThatFailsAtItsBootKeepsItsBirthFailedThere(t *testing.T) {
	b := failedAtBoot(t)
	rec, err := b.store.Get(context.Background(), "web")
	if err != nil {
		t.Fatal(err)
	}
	birth := rec.Birth
	if birth == nil || !birth.Failed || birth.Step != model.BornBooting || rec.Owner != nil {
		t.Fatalf("web's record keeps birth %+v, owner %+v; want it failed at booting, no process making it", birth, rec.Owner)
	}
	if !strings.HasPrefix(birth.Cause, "web did not boot") || birth.Sentence != "Failed at boot 2 of 4" {
		t.Errorf("web's failed birth says %q, %q; want its cause and where it failed", birth.Cause, birth.Sentence)
	}
	if want := []string{"boot: two", "boot: three", "boot: four"}; !slices.Equal(birth.LastLines, want) {
		t.Errorf("web's failed birth keeps lines %q; want its console's last %q", birth.LastLines, want)
	}
}

func TestAVMWhoseCreateFailedReadsNotCreatedAndOffersItsStart(t *testing.T) {
	b := failedAtBoot(t)
	vm, held := readVM(t, b, "web")
	if vm.State != model.Stopped || vm.Birth == nil || !vm.Birth.Failed || held != 0 {
		t.Fatalf("web reads %s, birth %+v, %d bytes held; want stopped, its birth failed, no memory held", vm.State, vm.Birth, held)
	}
	if s := model.StandingOf(vm); s != model.StandingNotCreated || s.Tone() != "bad" {
		t.Errorf("web stands %s, tone %s; want not-created, bad", s, s.Tone())
	}
	if head, _ := model.HeadOf(vm, b.mac.at); head.Sentence != "web was not created: it failed at boot." || !strings.HasPrefix(head.Line, "web did not boot") {
		t.Errorf("web's head says %q, %q; want where its create failed, then why", head.Sentence, head.Line)
	}
	if g := model.GesturesOf(vm); !slices.Equal(g, []model.Gesture{model.GestureStart}) {
		t.Errorf("web offers %v; want start, its create's retry", g)
	}
	if err := b.m.CheckDelete("web"); err != nil {
		t.Errorf("a delete of web, its create failed: %v; want it let through", err)
	}
}

func TestAStartThatAnswersTakesAFailedBirthOff(t *testing.T) {
	b := failedAtBoot(t)
	b.engine.starts = true
	if err := b.m.Start(context.Background(), "web"); err != nil {
		t.Fatalf("start of web: %v", err)
	}
	if rec, err := b.store.Get(context.Background(), "web"); err != nil || rec.Birth != nil {
		t.Errorf("web, started, keeps birth %+v, %v; want none", rec.Birth, err)
	}
}

func TestAStartThatFailsAgainKeepsTheBirthFailed(t *testing.T) {
	b := failedAtBoot(t)
	if err := b.m.Start(context.Background(), "web"); err == nil {
		t.Fatal("start of web, its engine running nothing, answered")
	}
	rec, err := b.store.Get(context.Background(), "web")
	if err != nil || rec.Birth == nil || !rec.Birth.Failed || rec.Birth.Step != model.BornBooting {
		t.Errorf("web, its start failed, keeps birth %+v, %v; want it failed at booting", rec.Birth, err)
	}
}

func TestAnInterruptedCreateKeepsNoFailedBirth(t *testing.T) {
	b := creating(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	b.engine.run = cancel
	if _, err := b.m.Create(ctx, model.CreateRequest{Image: "base-ubuntu", Name: "web", Lifetime: model.Persistent}, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v; want it interrupted", err)
	}
	if rec, err := b.store.Get(context.Background(), "web"); err != nil || rec.Birth != nil {
		t.Errorf("web, its create interrupted at its boot, keeps birth %+v, %v; want none", rec.Birth, err)
	}
}
