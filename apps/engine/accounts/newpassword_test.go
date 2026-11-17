package accounts_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

func capturing(t *testing.T) accounts.Program {
	t.Helper()
	for _, p := range accounts.Programs() {
		if p.Captures() {
			return p
		}
	}
	t.Fatal("no program captures")
	return ""
}

func bare(t *testing.T) accounts.Program {
	t.Helper()
	for _, p := range accounts.Programs() {
		if !p.Captures() {
			return p
		}
	}
	t.Fatal("every program captures")
	return ""
}

func TestTheSheetOffersASignInOnlyWhenThisMacHoldsItAndItsTokenCanBeTaken(t *testing.T) {
	kept, other := capturing(t), bare(t)
	signed := []accounts.ProgramRow{
		{Program: kept, State: accounts.ProgramSignedIn, Who: "docs-bot", How: accounts.HowOwn},
		{Program: other, State: accounts.ProgramSignedIn, Who: "docs-bot", How: accounts.HowOwn},
	}
	offer := accounts.NewPassword("this-mac", " / ", signed)
	if len(offer.Captures) != 1 || offer.Captures[0].Program != kept {
		t.Fatalf("captures %+v; want %s alone", offer.Captures, kept)
	}
	got := offer.Captures[0]
	if got.Label != kept.Label()+" / docs-bot" || got.Kind != kept.CaptureKind() || got.KindSentence != "Kept as "+got.Kind.Phrase()+", the kind "+string(kept)+" holds" || offer.CaptureLine != "" {
		t.Errorf("capture %+v, line %q", got, offer.CaptureLine)
	}
	granted := accounts.NewPassword("this-mac", " / ", []accounts.ProgramRow{{Program: kept, State: accounts.ProgramSignedIn, Who: "x", How: accounts.HowGranted}})
	if len(granted.Captures) != 0 || granted.CaptureLine == "" {
		t.Errorf("a granted sign-in is offered: %+v", granted)
	}
}

func TestTheSheetNamesTheKindsEveryProgramTakesTyped(t *testing.T) {
	offer := accounts.NewPassword("this-mac", " / ", nil)
	if len(offer.Programs) != len(accounts.Programs()) {
		t.Fatalf("%d programs; want %d", len(offer.Programs), len(accounts.Programs()))
	}
	for i, p := range offer.Programs {
		if p.Program != accounts.Programs()[i] || len(p.Kinds) == 0 || p.Kind != p.Kinds[0] || p.Label == "" || slices.Contains(p.Kinds, "") {
			t.Errorf("%+v", p)
		}
	}
	if offer.Intro != "It is kept in this-mac's Keychain; you can then grant it to a machine." || offer.CaptureLine != "Sign in to a program on this Mac first; then its sign-in can be kept" {
		t.Errorf("intro %q, line %q", offer.Intro, offer.CaptureLine)
	}
}

func TestALoginStepIsTitledByTheProgramAndTheMachine(t *testing.T) {
	if got := accounts.LoginTitle(capturing(t), "linux-1"); got != "Sign in to "+string(capturing(t))+" on linux-1" {
		t.Errorf("%q", got)
	}
}

func TestADeviceCodeStepLineSaysWhereToEnterItAndWhoSignsIn(t *testing.T) {
	line := capturing(t).StepLine(accounts.StepDeviceCode, "github.com", "linux-1")
	if line != "Enter this code at github.com. linux-1 signs in by itself once you approve it." {
		t.Errorf("%q", line)
	}
}
