package engine

import (
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	ReleaseEvery = 6 * time.Hour    // value: how long a reading of the latest release stands before the engine asks GitHub again
	ReleaseRetry = 30 * time.Minute // value: how long a refused ask waits before the next
)

// value: a release as GitHub gives it: its tag, its page and the names of
// its assets; the zero value is no release.

type Tagged struct {
	Tag, URL string
	Assets   []string
}

// value: what the engine keeps of its last asks: when GitHub last answered
// and the latest release it took, and the last ask it refused.

type ReleaseKept struct {
	Read    time.Time
	Latest  *Tagged
	Refused *ReleaseRefusal
}

type ReleaseRefusal struct {
	At      time.Time
	Message string
}

const notesRoot = "https://github.com/"

func ReleaseAsset(arch string) string {
	// value: the name of the binary a release carries for a Mac of arch.
	return "station-darwin-" + arch
}

func (p Tagged) Carries(arch string) bool {
	// invariant: a release is taken only once it carries this Mac's binary:
	// between its making and the attach of its darwin assets, it does not.
	if p.Tag == "" {
		return false
	}
	for _, a := range p.Assets {
		if a == ReleaseAsset(arch) {
			return true
		}
	}
	return false
}

func (k ReleaseKept) Answered(now time.Time, p Tagged, arch string) ReleaseKept {
	// invariant: an answer of GitHub moves the latest release only to one
	// that carries this Mac's binary and whose tag Newer reads; the previous
	// one stays otherwise. Its notes are kept only as a page of GitHub.
	out := ReleaseKept{Read: now, Latest: k.Latest}
	if p.Carries(arch) && model.Released(p.Tag) {
		out.Latest = &Tagged{Tag: p.Tag}
		if strings.HasPrefix(p.URL, notesRoot) {
			out.Latest.URL = p.URL
		}
	}
	return out
}

func (k ReleaseKept) Failed(now time.Time, message string) ReleaseKept {
	k.Refused = &ReleaseRefusal{At: now, Message: message}
	return k
}

func (k ReleaseKept) Due(now time.Time) time.Time {
	// invariant: the next ask is ReleaseEvery after the last answer, or
	// ReleaseRetry after a refusal that came after it; a kept ask ahead of
	// the clock is due now.
	next := k.Read.Add(ReleaseEvery)
	last := k.Read
	if k.Refused != nil && k.Refused.At.After(k.Read) {
		next, last = k.Refused.At.Add(ReleaseRetry), k.Refused.At
	}
	if now.Before(last) {
		return now
	}
	return next
}

func ReleaseOf(installed string, k ReleaseKept, kept bool) model.Release {
	out := model.Release{Installed: installed}
	if kept {
		if !k.Read.IsZero() {
			read := k.Read
			out.Checked = &read
		}
		if k.Latest != nil {
			out.Latest, out.Notes = k.Latest.Tag, k.Latest.URL
		}
		if k.Refused != nil && k.Refused.At.After(k.Read) {
			out.Refused = &model.ReleaseRefused{At: k.Refused.At, Message: k.Refused.Message}
		}
	}
	out.Behind = model.Newer(out.Latest, installed)
	out.Head = model.ReleaseHead(installed, out.Latest)
	return out
}
