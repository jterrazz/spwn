package apps

import "github.com/jterrazz/studio-station/src/catalog"

type Report struct {
	App     catalog.App
	Method  catalog.Method
	State   catalog.State
	Version string
	// value: the version an update would bring, when the method names it.
	Newest  string
	Managed bool
	Detail  string
	Live    string
	Pending bool
	// value: station cannot uninstall it here: its method removes nothing,
	// or the App Store installed it.
	Irreversible bool
	// value: for a manager there, what its methods hold that no app of the
	// catalog placed here installs, by key: the owner's own packages; and
	// whether a list of them did not answer, so what it holds is unknown.
	Holds       []string
	HoldsUnread bool
	// value: why the state is not ok, in a word a page can judge; Needs names
	// the catalogue tool whose absence explains it.
	Why   Why
	Needs string
}

const AlreadyInstalled = "already installed"

type Why string

const (
	WhyMissing   Why = "missing"
	WhyUnmanaged Why = "unmanaged"
	WhyOutdated  Why = "outdated"
	WhyPartial   Why = "partial"
	WhyLate      Why = "late"
	WhyFailing   Why = "failing"
	// value: the verb was interrupted before its end; its words are the
	// surface's.
	WhyInterrupted Why = "interrupted"
)

type Outcome struct {
	Name   string
	Verb   string
	Result catalog.State
	Detail string
	Steps  []Step
	// value: the method that holds the app, and the version it holds once
	// the verb has run ("" where nothing lists one).
	Method  catalog.Method
	Version string
	// value: why it ended broken, when the reason is typed rather than
	// worded in Detail.
	Why Why
	// value: the apps of the run that require it, for an app the run brought
	// in; none for an app named.
	RequiredBy []string
	// value: why it came, in the words of the binary that prints it, filled
	// from RequiredBy there.
	Reasons []string
	// value: the log that keeps its installer's words, as the owner reads
	// it (`~/.local/state/station/logs/<app>.log`), set when the verb failed.
	Log string
	// value: the folders its install just put on station's search path,
	// which a shell opened before it does not search.
	Path []string
}

type Step struct {
	Item   string
	Result catalog.State
	Detail string
	Why    Why
}

type InstallRequest struct {
	Names       []string
	Interactive bool
}

type UpgradeRequest struct {
	Names []string
	All   bool
	Via   []catalog.Manager
}

// invariant: a step's progress detail is typed lines, worded by the binary
// that prints them: `required-by gh, ghostty`, then `needs sudo` per need.
const (
	ReasonRequiredBy = "required-by "
	ReasonNeeds      = "needs "
)
