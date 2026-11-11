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
	// value: the log that keeps its installer's words, as the owner reads
	// it (`~/.local/state/station/logs/<app>.log`), set when the verb failed.
	Log string
	// value: the apps of the run that require it, for an app the run brought
	// in; none for an app named.
	RequiredBy []string
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
