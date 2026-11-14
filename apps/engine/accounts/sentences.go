package accounts

type Sentence string

const (
	SentenceSignInsAll        Sentence = "sign-ins-all"
	SentenceSignInsNone       Sentence = "sign-ins-none"
	SentenceSignInsSome       Sentence = "sign-ins-some"
	SentencePasswordsNone     Sentence = "passwords-none"
	SentencePasswordsSome     Sentence = "passwords-some"
	SentenceAgentReach        Sentence = "agent-reach"
	SentenceCaptureNote       Sentence = "capture-note"
	SentenceTermRule          Sentence = "term-rule"
	SentenceRowSignedByItself Sentence = "row-signed-by-itself"
	SentenceRowFromPassword   Sentence = "row-from-password"
)

func (s Sentence) Text() string {
	// value: the engine's wording of a sentence a page shows; a %d or %s is
	// filled by the one who knows the count or the name.
	switch s {
	case SentenceSignInsAll:
		return "Signed in to every program"
	case SentenceSignInsNone:
		return "Signed in to no program"
	case SentenceSignInsSome:
		return "Signed in to %d of %d programs"
	case SentencePasswordsNone:
		return "No password kept yet"
	case SentencePasswordsSome:
		return "%d kept, %d granted"
	case SentenceAgentReach:
		return "Only its terminal sees it"
	case SentenceCaptureNote:
		return "Station takes the token this sign-in left on this Mac and keeps it in the Keychain. The sign-in itself stays as it is."
	case SentenceTermRule:
		return "An agent keeps it up to 7 days. A machine may keep it until you remove it."
	case SentenceRowSignedByItself:
		return "signed in by itself"
	case SentenceRowFromPassword:
		return "from the password %s"
	}
	return string(s)
}
