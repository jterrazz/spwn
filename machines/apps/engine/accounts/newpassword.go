package accounts

import (
	"fmt"
	"strings"
)

type Capture struct {
	Program Program
	Who     string
	Scopes  []string `json:",omitempty"`
	Kind    KeyKind
	// value: how the sign-in is kept, in the kind its program holds.
	KindSentence string
	// value: the label proposed for the password, from the program and who it is.
	Label string
}

type TypedProgram struct {
	Program Program
	Label   string
	// value: the kinds a typed password of the program may be.
	Kinds []KeyKind
	// value: the kind offered first.
	Kind KeyKind
}

type VaultOffer struct {
	// value: where the password is kept and what can be done with it after.
	Intro string
	// value: one row per program this Mac is signed in to whose sign-in can be taken.
	Captures []Capture
	// invariant: set when Captures is empty and only then, so the way stays shown and says why it is bare.
	CaptureLine string
	// value: every program a password can be typed for, in the catalog's order.
	Programs []TypedProgram
}

func (k KeyKind) Phrase() string {
	switch k {
	case KeyToken:
		return "a token"
	case KeyFineGrained:
		return "a fine-grained token"
	case KeyAPIKey:
		return "an API key"
	case KeyOAuth:
		return "an OAuth token"
	case KeyPassword:
		return "a password"
	}
	return string(k)
}

func NewPassword(mac, separator string, signed []ProgramRow) VaultOffer {
	// invariant: separator is the glyph between a program and who, handed in
	// by the wire since a glyph is a ui token; a sign-in is offered for capture when this Mac holds it
	// itself and its program leaves a token to take; the sheet's every line
	// is worded here, so every adapter answers alike.
	offer := VaultOffer{
		Intro:    fmt.Sprintf(SentenceKeptIn.Text(), mac),
		Captures: []Capture{},
		Programs: make([]TypedProgram, 0, len(Programs())),
	}
	for _, row := range signed {
		if row.State != ProgramSignedIn || row.How == HowGranted || !row.Program.Captures() {
			continue
		}
		kind := row.Program.CaptureKind()
		label := row.Program.Label()
		if row.Who != "" {
			label += separator + row.Who
		}
		offer.Captures = append(offer.Captures, Capture{Program: row.Program, Who: row.Who, Scopes: row.Scopes, Kind: kind, KindSentence: fmt.Sprintf(SentenceKeptAs.Text(), kind.Phrase(), row.Program), Label: label})
	}
	if len(offer.Captures) == 0 {
		offer.CaptureLine = SentenceNoCapture.Text()
	}
	for _, p := range Programs() {
		kinds := p.Kinds()
		offer.Programs = append(offer.Programs, TypedProgram{Program: p, Label: p.Label(), Kinds: kinds, Kind: kinds[0]})
	}
	return offer
}

func BlankLabel(label string) bool { return strings.TrimSpace(label) == "" }
