package machinefile

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/page"
)

// value: the method column of a block's row: what writes it is a shell
// block, whatever manager installed its item.
const blockMethod = "shell"

// value: where a block lives, as the plan and the run say it: the file and
// the block's id.
func blockWhere(b machinefile.Block) string { return "~/.zshrc [" + b.ID + "]" }

func lines(body string) string {
	n := strings.Count(body, "\n")
	if n == 1 {
		return "1 line"
	}
	return strconv.Itoa(n) + " lines"
}

func blockRow(b machinefile.Block, words ...string) page.Row {
	row := page.Row{ID: b.ID, Label: b.Item, Cells: []page.Value{page.Texts(blockMethod)}}
	switch b.Condition {
	case machinefile.Present:
		row.Mark = page.OK
	case machinefile.Changed:
		row.Mark = page.Attention
	case machinefile.Extra:
		row.Mark = page.NotKnown
	default:
		row.Mark = page.Absent
	}
	row.Detail = page.Texts(words...)
	return row
}

// value: the rows of the blocks apply does one thing with: set (written
// where it is absent, or written anew over what station last wrote), or left
// (edited by hand since, said with the way to see how).
func blockRows(p machinefile.Plan, a machinefile.Action) []page.Row {
	var rows []page.Row
	for _, b := range p.Blocks {
		switch {
		case b.Action != a || b.Condition == machinefile.Extra:
		case a == machinefile.Set && b.Condition == machinefile.Outdated:
			rows = append(rows, blockRow(b, blockWhere(b)+": "+lines(b.Zsh), "station's earlier text, written anew"))
		case a == machinefile.Set:
			rows = append(rows, blockRow(b, blockWhere(b)+": "+lines(b.Zsh)))
		case a == machinefile.Leave:
			rows = append(rows, blockRow(b, blockWhere(b), "edited by hand: station plan --diff shows how"))
		}
	}
	return rows
}

// value: the rows of the blocks station wrote for an extra, each with what
// `apply --remove` does with it: takes it with its app, keeps it with an
// app that stays, or lets go of it as the owner's.
func extraBlockRows(p machinefile.Plan, lead ...string) []page.Row {
	var rows []page.Row
	for _, b := range p.Blocks {
		if b.Condition == machinefile.Extra {
			rows = append(rows, blockRow(b, append(slices.Clone(lead), blockWhere(b), blockFate(b))...))
		}
	}
	return rows
}

func blockFate(b machinefile.Block) string {
	switch b.Fate {
	case machinefile.Taken, machinefile.Gone:
		return "written by station, goes with " + b.Item
	case machinefile.KeptEdited:
		return "edited by hand since station wrote it: kept, and yours"
	case machinefile.KeptIrreversible, machinefile.KeptAnotherWay:
		return "kept with " + b.Item + ", and yours"
	case machinefile.KeptRequired, machinefile.KeptHolding, machinefile.KeptManagerMissing, machinefile.KeptUnread:
	}
	return "kept with " + b.Item
}
