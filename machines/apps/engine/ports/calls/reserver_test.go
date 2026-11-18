package calls

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type audienceEngine struct {
	version    string
	keeps      bool
	helloErr   error
	shareAs    catalog.Audience
	releaseErr error
	reserved   []catalog.Audience
	released   int
	asked      int
}

func (e *audienceEngine) MacPorts(context.Context) (ports.MacPorts, error) {
	return ports.MacPorts{}, nil
}

func (e *audienceEngine) Reserve(_ context.Context, vm string, port int, to catalog.Audience) (ports.SharedPort, error) {
	e.reserved = append(e.reserved, to)
	return ports.SharedPort{VM: vm, GuestPort: port, To: e.shareAs}, nil
}

func (e *audienceEngine) Release(_ context.Context, vm string, port int) (ports.Reservation, error) {
	e.released++
	return ports.Reservation{VM: vm, GuestPort: port}, e.releaseErr
}

func (e *audienceEngine) KeepsAudience(context.Context) (version string, keeps bool, err error) {
	e.asked++
	return e.version, e.keeps, e.helloErr
}

func reserverOn(e *audienceEngine) *Reserver {
	return NewReserver(Stores{}, e, ReserverOptions{ThisMachine: catalog.Mac, ThisMacName: "mac-mini"})
}

func refusal(t *testing.T, err error, code fail.Code) *fail.Error {
	t.Helper()
	var f *fail.Error
	if !errors.As(err, &f) || f.Code != code {
		t.Fatalf("err = %v, want the refusal %s", err, code)
	}
	return f
}

func TestAnAudienceIsNeverSentToAnEngineThatDoesNotReadOne(t *testing.T) {
	e := &audienceEngine{version: "v3.2.0", shareAs: catalog.AudiencePublic}
	_, err := reserverOn(e).Share(t.Context(), "linux-4", 3000, catalog.AudiencePrivate)
	f := refusal(t, err, ports.CodeEngineTooOld)
	if f.Subject != "mac-mini" || f.Detail["version"] != "v3.2.0" || f.Type != fail.FarTooOld {
		t.Fatalf("refusal = %+v, want the engine of mac-mini at v3.2.0, far-too-old", f)
	}
	if len(e.reserved) != 0 {
		t.Fatalf("reserved %v, want nothing sent to the engine", e.reserved)
	}
}

func TestAnEngineThatDoesNotAnswerItsHelloIsDownBeforeAnyShare(t *testing.T) {
	e := &audienceEngine{helloErr: ports.ErrEngineDown}
	_, err := reserverOn(e).Share(t.Context(), "linux-4", 3000, catalog.AudienceLocal)
	if !errors.Is(err, ports.ErrEngineDown) || len(e.reserved) != 0 {
		t.Fatalf("err = %v, reserved %v; want the engine down and nothing sent", err, e.reserved)
	}
}

func TestAShareWithoutAudienceAsksNoAbility(t *testing.T) {
	e := &audienceEngine{version: "v3.2.0", shareAs: catalog.AudiencePublic}
	v, err := reserverOn(e).Share(t.Context(), "linux-4", 3000, "")
	if err != nil || v.To != catalog.AudiencePublic || e.asked != 0 || e.released != 0 {
		t.Fatalf("share = %+v, %v; asked %d, released %d: want the rights' share, no hello asked", v, err, e.asked, e.released)
	}
}

func TestAnEngineThatReadsTheAudienceIsSentIt(t *testing.T) {
	e := &audienceEngine{keeps: true, shareAs: catalog.AudiencePrivate}
	v, err := reserverOn(e).Share(t.Context(), "linux-4", 3000, catalog.AudiencePrivate)
	if err != nil || v.To != catalog.AudiencePrivate || len(e.reserved) != 1 || e.reserved[0] != catalog.AudiencePrivate || e.released != 0 {
		t.Fatalf("share = %+v, %v; reserved %v, released %d", v, err, e.reserved, e.released)
	}
}

func TestAShareThatComesBackWithAnotherAudienceIsReleasedAndRefused(t *testing.T) {
	e := &audienceEngine{keeps: true, shareAs: catalog.AudiencePublic}
	v, err := reserverOn(e).Share(t.Context(), "linux-4", 3000, catalog.AudiencePrivate)
	f := refusal(t, err, ports.CodeAudienceNotKept)
	if e.released != 1 || f.Detail["released"] != "yes" || f.Detail["got"] != "public" || f.Detail["asked"] != "private" || v.To != "" {
		t.Fatalf("refusal = %+v, share %+v, released %d; want it released, public named against private", f, v, e.released)
	}
}

func TestAShareThatCannotBeReleasedSaysItIsStillShared(t *testing.T) {
	e := &audienceEngine{keeps: true, shareAs: catalog.AudiencePublic, releaseErr: errors.New("release failed")}
	_, err := reserverOn(e).Share(t.Context(), "linux-4", 3000, catalog.AudiencePrivate)
	f := refusal(t, err, ports.CodeAudienceNotKept)
	if e.released != 1 || f.Detail["released"] != "" || f.Detail["cause"] != "release failed" {
		t.Fatalf("refusal = %+v, released %d; want a release tried and the share said still on", f, e.released)
	}
}
