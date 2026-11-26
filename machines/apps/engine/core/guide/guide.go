package guide

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

type Status string

// value: where a step of an act stands, as the window's rail draws it.
const (
	Done    Status = "done"
	Now     Status = "now"
	Waiting Status = "waiting"
	Failed  Status = "failed"
	Next    Status = "next"
)

func (Status) Enum() []string {
	return []string{string(Done), string(Now), string(Waiting), string(Failed), string(Next)}
}

func (s Status) Word() string {
	switch s {
	case Done:
		return "Done"
	case Now:
		return "Now"
	case Waiting:
		return "Waiting for you"
	case Failed:
		return "Failed"
	case Next:
		return "Next"
	}
	return string(s)
}

type Move string

// value: whose move the act waits on: the engine works, or the owner acts.
const (
	MoveEngine Move = "engine"
	MoveOwner  Move = "owner"
)

func (Move) Enum() []string { return []string{string(MoveEngine), string(MoveOwner)} }

func (m Move) Word() string {
	if m == MoveOwner {
		return "Your move"
	}
	return "Spwn works"
}

type Picture string

// value: the drawing a step shows, one per moment of an act; the engine
// names it, the window draws it, and both lists are one.
const (
	PictureInstallEngine  Picture = "install-engine"
	PictureFetchImage     Picture = "fetch-image"
	PictureBuild1         Picture = "build-1"
	PictureBuild2         Picture = "build-2"
	PictureBuild3         Picture = "build-3"
	PictureBuild4         Picture = "build-4"
	PictureBuildFailed    Picture = "build-failed"
	PictureBoot           Picture = "boot"
	PictureBootFailed     Picture = "boot-failed"
	PictureApplyFile      Picture = "apply-file"
	PictureSystemSettings Picture = "system-settings"
	PictureTouchID        Picture = "touch-id"
	PictureMacSearch      Picture = "mac-search"
	PictureMacLink        Picture = "mac-link"
	PictureMacJoined      Picture = "mac-joined"
	PictureDeviceKey      Picture = "device-key"
	PictureDone           Picture = "done"
)

func Pictures() []Picture {
	return []Picture{
		PictureInstallEngine, PictureFetchImage, PictureBuild1, PictureBuild2, PictureBuild3, PictureBuild4, PictureBuildFailed,
		PictureBoot, PictureBootFailed, PictureApplyFile, PictureSystemSettings, PictureTouchID, PictureMacSearch, PictureMacLink,
		PictureMacJoined, PictureDeviceKey, PictureDone,
	}
}

func (Picture) Enum() []string {
	all := Pictures()
	out := make([]string, len(all))
	for i, p := range all {
		out[i] = string(p)
	}
	return out
}

func (p Picture) Word() string {
	switch p {
	case PictureInstallEngine:
		return "The engine being installed"
	case PictureFetchImage:
		return "An image being fetched"
	case PictureBuild1, PictureBuild2, PictureBuild3, PictureBuild4:
		return "An image being built"
	case PictureBuildFailed:
		return "An image whose build failed"
	case PictureBoot:
		return "A machine starting"
	case PictureBootFailed:
		return "A machine that did not start"
	case PictureApplyFile:
		return "A Machinefile being applied"
	case PictureSystemSettings:
		return "System Settings"
	case PictureTouchID:
		return "Touch ID"
	case PictureMacSearch:
		return "A Mac being looked for"
	case PictureMacLink:
		return "A Mac being linked"
	case PictureMacJoined:
		return "A Mac that joined"
	case PictureDeviceKey:
		return "This Mac's key"
	case PictureDone:
		return "Done"
	}
	return string(p)
}

type ProgressKind string

const (
	// value: how far a step is: its last live line; a measure is added only
	// where the engine measures one.
	Live ProgressKind = "live"
)

func (ProgressKind) Enum() []string { return []string{string(Live)} }

func (ProgressKind) Word() string { return "Live" }

type Ended string

// value: how an act ended: made, failed at its step, or cut by the engine
// stopping under it.
const (
	EndedDone        Ended = "done"
	EndedFailed      Ended = "failed"
	EndedInterrupted Ended = "interrupted"
)

func (Ended) Enum() []string {
	return []string{string(EndedDone), string(EndedFailed), string(EndedInterrupted)}
}

func (e Ended) Word() string {
	switch e {
	case EndedDone:
		return "Done"
	case EndedFailed:
		return "Failed"
	case EndedInterrupted:
		return "Interrupted"
	}
	return string(e)
}

type Step struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type Headline struct {
	Now  string `json:"now"`
	Next string `json:"next"`
}

type Progress struct {
	Kind ProgressKind `json:"kind"`
	Line string       `json:"line"`
}

type Prompt struct {
	// value: the engine's lead, then the sentence macOS shows, quoted.
	Lead  string `json:"lead"`
	Quote string `json:"quote"`
}

type Gesture struct {
	// invariant: exactly one of Call and Nav: a call and its args, each a
	// field of the call's input (rule fix-args-are-inputs), or the page it
	// opens; never a command, the window runs none.
	Label   string            `json:"label"`
	Call    string            `json:"call,omitempty"`
	Args    map[string]string `json:"args,omitempty"`
	On      string            `json:"on,omitempty"`
	Confirm string            `json:"confirm,omitempty"`
	Nav     string            `json:"nav,omitempty"`
}

type View struct {
	Illustration Picture   `json:"illustration"`
	Headline     Headline  `json:"headline"`
	Sentence     string    `json:"sentence"`
	Move         Move      `json:"move"`
	Progress     *Progress `json:"progress,omitempty"`
	Prompt       *Prompt   `json:"prompt,omitempty"`
	Kept         string    `json:"kept,omitempty"`
	Action       *Gesture  `json:"action,omitempty"`
	Quiet        *Gesture  `json:"quiet,omitempty"`
}

type Guide struct {
	// value: the act as the window draws it, all of it the engine's words:
	// its id `<family>:<subject>`, its rail, the current step's view, how
	// it ended, the one line a list shows, and whether it still runs.
	ID      string `json:"id"`
	Act     string `json:"act"`
	Subject string `json:"subject"`
	Steps   []Step `json:"steps"`
	View    View   `json:"view"`
	Ended   Ended  `json:"ended,omitempty"`
	Line    string `json:"line"`
	Live    bool   `json:"live"`
}

type Guides struct {
	Guides []Guide `json:"guides"`
}

type Rung struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Act struct {
	// value: what the engine keeps of a guided call as it runs: its id, its
	// family and subject, the steps it declared, the one it reached and its
	// last line, and, once it ended, how, with its failure's code and
	// words; At is when it reached its step or ended.
	ID      string    `json:"id"`
	Family  string    `json:"family"`
	Subject string    `json:"subject"`
	Rail    []Rung    `json:"rail"`
	Step    string    `json:"step"`
	Line    string    `json:"line,omitempty"`
	Ended   Ended     `json:"ended,omitempty"`
	Code    string    `json:"code,omitempty"`
	Cause   string    `json:"cause,omitempty"`
	At      time.Time `json:"at"`
}

type Acts struct {
	Acts []Act `json:"acts"`
}

func ID(family, subject string) string { return family + ":" + subject }

type Phase struct {
	// value: a step a guided call reached: the subject it settled, the
	// steps it declares (its first phase), the step and its live line.
	Subject string
	Rail    []Rung
	Step    string
	Line    string
}

type Phaser interface {
	Phase(Phase)
}

func Report(p call.Progress, ph Phase) {
	// invariant: a phase reaches the engine's board only through a progress
	// that keeps one; any other progress (the CLI's) hears its Step alone.
	if r, ok := p.(Phaser); ok {
		r.Phase(ph)
	}
}

func Rail(a Act) []Step {
	// invariant: the steps before the one reached are done; that one runs
	// now, failed when the act failed or was cut there, done when it ended
	// made; the steps after it are next, all done once it is made.
	at := slices.IndexFunc(a.Rail, func(r Rung) bool { return r.ID == a.Step })
	out := make([]Step, len(a.Rail))
	for i, r := range a.Rail {
		s := Step{ID: r.ID, Label: r.Label, Status: Next}
		switch {
		case a.Ended == EndedDone, i < at:
			s.Status = Done
		case i == at && a.Ended == "":
			s.Status = Now
		case i == at:
			s.Status = Failed
		}
		out[i] = s
	}
	return out
}
