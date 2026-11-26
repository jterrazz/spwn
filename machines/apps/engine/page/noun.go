package page

type Noun string

// value: what a page counts that no chapter of a Machinefile names: a
// client says a number of them with the noun's One and Other, and the
// listeners a Mac's ports left out for living a short while.
const (
	NounMachine       Noun = "machine"
	NounSkill         Noun = "skill"
	NounListener      Noun = "listener"
	NounQuietListener Noun = "quiet-listener"
	NounRestart       Noun = "restart"
	NounLine          Noun = "line"
	NounPackage       Noun = "package"
	NounApp           Noun = "app"
	NounSetting       Noun = "setting"
	NounPassword      Noun = "password"
	NounGrant         Noun = "grant"
	NounBlock         Noun = "block"
	NounWikiPage      Noun = "wiki-page"
	NounRow           Noun = "row"
	NounWikiSeat      Noun = "wiki-seat"
	NounWikiSection   Noun = "wiki-section"
	NounWikiFinding   Noun = "wiki-finding"
)

func (Noun) Enum() []string {
	return []string{string(NounMachine), string(NounSkill), string(NounListener), string(NounQuietListener), string(NounRestart), string(NounLine), string(NounPackage), string(NounApp), string(NounSetting), string(NounPassword), string(NounGrant), string(NounBlock), string(NounWikiPage), string(NounRow), string(NounWikiSeat), string(NounWikiSection), string(NounWikiFinding)}
}
