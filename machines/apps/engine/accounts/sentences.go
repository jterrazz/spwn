package accounts

type Sentence string

const (
	SentenceSignInsAll        Sentence = "sign-ins-all"
	SentenceSignInsNone       Sentence = "sign-ins-none"
	SentenceSignInsSome       Sentence = "sign-ins-some"
	SentencePasswordsNone     Sentence = "passwords-none"
	SentencePasswordsSome     Sentence = "passwords-some"
	SentenceCaptureNote       Sentence = "capture-note"
	SentenceRowSignedByItself Sentence = "row-signed-by-itself"
	SentenceRowFromPassword   Sentence = "row-from-password"
	SentenceRemoveEnds        Sentence = "remove-ends"
	SentenceRemoveNothing     Sentence = "remove-nothing"
	SentenceKeptIn            Sentence = "kept-in"
	SentenceKeptAs            Sentence = "kept-as"
	SentenceNoCapture         Sentence = "no-capture"
	SentenceTypedWay          Sentence = "typed-way"
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
	case SentenceCaptureNote:
		return "Machines takes the token this sign-in left on this Mac and keeps it in the Keychain. The sign-in itself stays as it is."
	case SentenceRowSignedByItself:
		return "signed in by itself"
	case SentenceRowFromPassword:
		return "from the password %s"
	case SentenceRemoveEnds:
		return "Its grants end: %s"
	case SentenceRemoveNothing:
		return "Nothing else ends"
	case SentenceKeptIn:
		return "It is kept in %s's Keychain; you can then grant it to a machine."
	case SentenceKeptAs:
		return "Kept as %s, the kind %s holds"
	case SentenceNoCapture:
		return "Sign in to a program on this Mac first; then its sign-in can be kept"
	case SentenceTypedWay:
		return "A token you made yourself"
	}
	return string(s)
}
