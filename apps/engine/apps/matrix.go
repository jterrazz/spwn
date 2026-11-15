package apps

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type Column struct {
	settings.LivesOn
	// value: when the machine answered; zero for a machine that was not read.
	Read time.Time
	// value: why a machine that should have answered did not, in the owner's words.
	Why string
}

type Cell struct {
	// value: the app's state on one machine; Not known where the machine was
	// not read.
	State State
	// value: the version it holds there, and the one an update would bring.
	Version string
	Newest  string
	// value: the app does not live on that machine: its list does not hold it.
	NotHere bool
}

type MatrixRow struct {
	Name  string
	Cells []Cell
	// value: the machines that carry it, of those read that it lives on.
	Carry, Of int
}

type Matrix struct {
	Columns []Column
	Servers []string
	Rows    []MatrixRow
}

type Reading struct {
	Reports []Report
	Read    bool
}

func MatrixOf(cols []Column, reads []Reading, servers []string) Matrix {
	// invariant: one row per app some machine carries, in the order the
	// machines list them, one cell per column; a machine not read says Not
	// known, one whose list lacks the app says it does not live there.
	m := Matrix{Columns: cols, Servers: servers}
	at := map[string]int{}
	for _, rd := range reads {
		for _, r := range rd.Reports {
			if _, ok := at[r.App.Name]; !ok {
				at[r.App.Name] = len(m.Rows)
				m.Rows = append(m.Rows, MatrixRow{Name: r.App.Name, Cells: slices.Repeat([]Cell{{State: NotKnown}}, len(cols))})
			}
		}
	}
	for i, rd := range reads {
		if !rd.Read {
			continue
		}
		listed := map[string]Report{}
		for _, r := range rd.Reports {
			listed[r.App.Name] = r
		}
		for j := range m.Rows {
			r, ok := listed[m.Rows[j].Name]
			if !ok {
				m.Rows[j].Cells[i] = Cell{State: NotInstalled, NotHere: true}
				continue
			}
			st := StateOf(r)
			m.Rows[j].Cells[i] = Cell{State: st, Version: r.Version, Newest: r.Newest}
			m.Rows[j].Of++
			if st.Installed() {
				m.Rows[j].Carry++
			}
		}
	}
	m.Rows = slices.DeleteFunc(m.Rows, func(r MatrixRow) bool { return r.Carry == 0 })
	return m
}
