package token

const (
	MarkOK        = "●"
	MarkAttention = "◐"
	MarkAbsent    = "○"
	MarkBroken    = "✗"
	MarkPending   = "◌"
	MarkOffHere   = "⊘"
	MarkNotFor    = "–"
	MarkUnknown   = "?"

	Arrow    = "→"
	Sep      = " · "
	Ellipsis = "…"
	Rule     = "─"
	Divider  = "│"
	TreeMid  = "├─ "
	TreeEnd  = "└─ "
	TreeBar  = "│  "
	Spinner  = "⣾⣽⣻⢿⡿⣟⣯⣷"
	Spark    = "▁▂▃▄▅▆▇█"
	Unknown  = "?"
	Cursor   = "▸"
	// value: the mark of a step that succeeded.
	Tick = "✓"
)

func Glyphs() []string {
	return []string{
		MarkOK, MarkAttention, MarkAbsent, MarkBroken, MarkPending, MarkOffHere, MarkNotFor, MarkUnknown,
		Arrow, "·", Ellipsis, Rule, Divider, "├", "└", Spinner, Spark, Cursor, Tick,
	}
}
