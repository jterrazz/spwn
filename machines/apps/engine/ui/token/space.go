package token

import "time"

const (
	Margin      = 1
	Indent      = 2
	Gap         = 2
	TreeStep    = 3
	MarkWidth   = 2
	LabelMin    = 8
	LabelMax    = 28
	DetailMin   = 20
	NarrowWidth = 80
	MinWidth    = 40

	SpinnerFrame = 80 * time.Millisecond
	WaitDelay    = 300 * time.Millisecond
)
