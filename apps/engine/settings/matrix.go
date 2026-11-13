package settings

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type Away string

const (
	// value: why a machine's settings cannot be read now: they can.
	Present Away = ""
	Stopped Away = "stopped"
	// value: the Mac does not answer, or the workspace's Mac does not.
	Silent Away = "silent"
)

type LivesOn struct {
	// value: one machine of All machines whose settings are read: a Mac, or
	// a workspace of one.
	Name        string
	Ref         machine.Ref
	ThisMachine bool
	Mac         bool
	Away        Away
	Answered    time.Time
}

type LastRead struct {
	// value: when a workspace's settings were last read, and what they were.
	At   time.Time `json:"at"`
	Rows []Row     `json:"rows"`
}

func LivesOnAll(f machines.AllMachines) (livesOn []LivesOn, servers []string) {
	// invariant: every Mac, then every workspace, those that run first; a
	// server is named apart, never a place: station does not manage its
	// settings.
	var macs, spaces []LivesOn
	for _, h := range f.Macs {
		silent := !h.Reachable && !h.ThisMac
		mac := LivesOn{Name: h.Name, Ref: machine.Ref{Name: h.Name}, ThisMachine: h.ThisMac, Mac: true}
		if silent {
			mac.Away, mac.Answered = Silent, h.Answered
		}
		macs = append(macs, mac)
		for _, vm := range h.VMs {
			if vm.Lifetime.Kind() == machine.Server {
				servers = append(servers, vm.Name)
				continue
			}
			p := LivesOn{Name: vm.Name, Ref: machine.Ref{Name: vm.Name, On: h.Name}}
			switch {
			case silent:
				p.Away, p.Answered = Silent, h.Answered
			case vm.State != machines.Running:
				p.Away = Stopped
			}
			spaces = append(spaces, p)
		}
	}
	slices.SortStableFunc(spaces, func(a, b LivesOn) int { return awayRank(a.Away) - awayRank(b.Away) })
	return append(macs, spaces...), servers
}

func awayRank(a Away) int {
	switch a {
	case Present:
		return 0
	case Stopped:
		return 1
	case Silent:
	}
	return 2
}

type Column struct {
	LivesOn
	// value: the rows read, now or, for a stopped workspace, at Read; none
	// when nothing could be read.
	Rows []Row
	Read time.Time
	// value: why a machine that should have answered did not.
	Why string
}

func (c Column) Known() bool { return c.Rows != nil }

type Cell string

const (
	// value: a setting's state on a machine that was not read; any other
	// cell is its Set by station or its checked state there.
	CellUnknown Cell = "unknown"
)

func CellOf(r Row) Cell {
	if r.Group == Checked && r.Checked != "" {
		return Cell(r.Checked)
	}
	return Cell(r.State)
}

func (c Cell) OK() bool { return c == Cell(StateApplied) || c == Cell(AsExpected) }

func (c Cell) Look() bool {
	// value: a cell the owner should look at: a setting changed since
	// station applied it, a check not as expected.
	return c == Cell(StateChangedSince) || c == Cell(NeedsFix)
}

func (c Cell) away() bool {
	return c == Cell(StateNotForThisMachine)
}

type MatrixRow struct {
	Name  string
	Group Group
	Cells []Cell
	// value: whether each cell is counted: not for its machine, or of the
	// Always-on Mac group on a Mac that has none of it, it is not.
	Counts []bool
	// value: the machines where it is OK, of those it lives on and that were
	// read: a cell not for its machine is left out.
	OK, Of int
}

type Matrix struct {
	Columns []Column
	Servers []string
	Rows    []MatrixRow
}

func MatrixOf(cols []Column, servers []string) Matrix {
	// invariant: one row per setting, in the order the machines list them,
	// one cell per column; the Always-on Mac group counts only on a Mac
	// where one of it is applied, as that Mac's own list counts it.
	m := Matrix{Columns: cols, Servers: servers}
	at := map[string]int{}
	for _, c := range cols {
		for _, r := range c.Rows {
			if _, ok := at[r.Name]; !ok {
				at[r.Name] = len(m.Rows)
				m.Rows = append(m.Rows, MatrixRow{Name: r.Name, Group: r.Group, Cells: slices.Repeat([]Cell{CellUnknown}, len(cols)), Counts: make([]bool, len(cols))})
			}
		}
	}
	for i, c := range cols {
		opted := touched(c.Rows)
		for _, r := range c.Rows {
			row := &m.Rows[at[r.Name]]
			cell := CellOf(r)
			row.Cells[i] = cell
			if cell.away() || (r.Group == AlwaysOnMac && !opted) {
				continue
			}
			row.Of++
			row.Counts[i] = true
			if cell.OK() {
				row.OK++
			}
		}
	}
	return m
}

func touched(rows []Row) bool {
	return slices.ContainsFunc(rows, func(r Row) bool {
		return r.Group == AlwaysOnMac && (r.State == StateApplied || r.State == StateChangedSince || r.State == StateOffThisMac)
	})
}

func (c Column) Row(name string) (Row, bool) {
	// value: the row of a column's setting, as that machine read it.
	i := slices.IndexFunc(c.Rows, func(r Row) bool { return r.Name == name })
	if i < 0 {
		return Row{}, false
	}
	return c.Rows[i], true
}
