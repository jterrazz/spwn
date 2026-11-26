package approvals

import (
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type ID string

// value: the approvals, each a consent macOS gives Spwn, which Spwn can
// ask for and read but never give itself (ADR-105); the table is closed.
const (
	LocalNetwork ID = "local-network"
)

func (ID) Enum() []string { return []string{string(LocalNetwork)} }

func (id ID) Word() string {
	if id == LocalNetwork {
		return "Local Network"
	}
	return string(id)
}

func (id ID) Sentence() string {
	// value: what the approval lets Spwn do, in the engine's words.
	if id == LocalNetwork {
		return "Spwn reaches the VMs it runs and your Macs nearby."
	}
	return ""
}

func (id ID) Required() bool {
	// value: whether part of Spwn's work stops without it: no VM is reached
	// without Local Network, and Spwn still runs.
	return id == LocalNetwork
}

func (id ID) States() []State {
	// value: the states the approval can read in, its closed list.
	if id == LocalNetwork {
		return []State{Approved, Refused, Checking, Unknown}
	}
	return nil
}

type Row struct {
	// value: one approval of a Mac: its state, the process macOS charges
	// with it, whether Spwn needs it, what it enables, and the gesture that
	// moves it, empty while none is owed.
	ID       ID          `json:"id"`
	State    State       `json:"state"`
	Holder   string      `json:"holder"`
	Required bool        `json:"required"`
	Sentence string      `json:"sentence"`
	Fix      finding.Fix `json:"fix"`
}

type Listing struct {
	// value: every approval of one Mac, in the table's order.
	Mac  string `json:"mac"`
	Rows []Row  `json:"rows"`
}

func ListingOf(mac string, read machines.LocalNetwork, bundled bool) Listing {
	// value: a Mac's approvals as its engine reads them: Local Network from
	// the engine's read, held by the app that carries the engine or by the
	// bare binary.
	return Listing{Mac: mac, Rows: []Row{localNetworkRow(mac, StateOf(read), bundled)}}
}

func localNetworkRow(mac string, s State, bundled bool) Row {
	// invariant: a refused row's fix is the pane, since macOS asks no more;
	// the others owe none until approvals.ask exists.
	row := Row{ID: LocalNetwork, State: s, Holder: machines.LocalNetworkApp(bundled), Required: LocalNetwork.Required(), Sentence: LocalNetwork.Sentence()}
	if s == Refused {
		row.Fix = finding.Fix{Path: machines.LocalNetworkFix(mac, bundled), Pane: catalog.PaneLocalNetwork}
	}
	return row
}
