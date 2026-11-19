package machines

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type Release struct {
	// value: the version of the engine that answers.
	Installed string
	// value: the latest release that carries this Mac's binary; absent
	// while none is known.
	Latest string `json:",omitempty"`
	Behind bool
	Head   verdict.Verdict
	// value: the page of the latest release, its notes.
	Notes string `json:",omitempty"`
	// value: when GitHub last answered; absent while it never has.
	Checked *time.Time `json:",omitempty"`
	// value: the last ask's failure, when it came after Checked.
	Refused *ReleaseRefused `json:",omitempty"`
}

type ReleaseRefused struct {
	At      time.Time
	Message string
}

func ReleaseHead(installed, latest string) verdict.Verdict {
	// invariant: a version Newer cannot read is a build of no release,
	// whatever the latest release is; a release past the latest one is
	// never behind it.
	switch {
	case !Released(installed):
		return verdict.Verdict{Tone: verdict.Plain, Sentence: "This Mac runs a build of no release."}
	case latest == "":
		return verdict.Verdict{Tone: verdict.Plain, Sentence: "The latest release is not known yet."}
	case Newer(latest, installed):
		return verdict.Verdict{Tone: verdict.New, Sentence: catalog.ReleaseName + " " + latest + " is out.", Line: "This Mac runs " + installed + "."}
	case Newer(installed, latest):
		return verdict.Verdict{Tone: verdict.Plain, Sentence: "This Mac runs " + installed + ", past the latest release " + latest + "."}
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: "This Mac runs the latest release."}
}
