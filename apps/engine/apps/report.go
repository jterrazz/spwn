package apps

import "github.com/jterrazz/studio-station/apps/engine/catalog"

type Report struct {
	App    catalog.App
	Method catalog.Method
	// value: the key `apps update --manager` reaches its method by: the
	// first its manager package installs by (cask's is brew), else its own.
	Manager string
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
	// value: the newest version the app's manager offers, whatever range
	// holds it, read by an update within a range ("" where unread).
	Newest string
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
	Package string
	Result  catalog.State
	Detail  string
	Why     Why
}

type InstallRequest struct {
	Names       []string
	Interactive bool
	// value: the method the apps are installed by and the version asked,
	// a Machinefile's step's; empty, the method this machine carries them
	// by, at its manager's version.
	Method  catalog.Method
	Version string
}

type UpgradeRequest struct {
	Names []string
	All   bool
	// value: the manager keys whose managers' apps are updated, each
	// reaching every key its manager package manages (brew reaches cask).
	Via []catalog.ManagerKey
	// value: the range a Machinefile lets the one app named move within
	// (`latest`, `24`, `=24.11.3`); given, a method that holds any range
	// installs it at the newest the range holds, and reads the newest its
	// manager offers past it.
	Within string
}

// invariant: a step's progress detail is typed lines, worded by the binary
// that prints them: `required-by gh, ghostty`, then `needs sudo` per need.
const (
	ReasonRequiredBy = "required-by "
	ReasonNeeds      = "needs "
)
