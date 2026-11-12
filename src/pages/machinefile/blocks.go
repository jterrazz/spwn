package machinefile

import (
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
// where it is absent), left (edited by hand since, said with the way to see
// how), or removed (its item is in the plan no more).
func blockRows(p machinefile.Plan, a machinefile.Action) []page.Row {
	var rows []page.Row
	for _, b := range p.Blocks {
		if b.Action != a {
			continue
		}
		switch a {
		case machinefile.Set:
			rows = append(rows, blockRow(b, blockWhere(b)+": "+lines(b.Zsh)))
		case machinefile.Leave:
			rows = append(rows, blockRow(b, blockWhere(b), "edited by hand: station plan --diff shows how"))
		default:
			rows = append(rows, blockRow(b, blockWhere(b), "its item is no longer listed"))
		}
	}
	return rows
}
