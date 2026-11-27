package ui

import "github.com/fatih/color"

var (
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	cyan   = color.New(color.FgCyan)
	dim    = color.New(color.Faint)
	bold   = color.New(color.Bold)
)

func check() string          { return green.Sprint("✓") }
func warn() string           { return yellow.Sprint("!") }
func cross() string          { return red.Sprint("✗") }
func faint(s string) string  { return dim.Sprint(Clean(s)) }
func strong(s string) string { return bold.Sprint(Clean(s)) }

// Exported color helpers for use outside the ui package; each colours
// Clean(s), so a project's text styled by one is safe to print.
func Faint(s string) string  { return dim.Sprint(Clean(s)) }
func Strong(s string) string { return bold.Sprint(Clean(s)) }
func Green(s string) string  { return green.Sprint(Clean(s)) }
func Yellow(s string) string { return yellow.Sprint(Clean(s)) }
func Red(s string) string    { return red.Sprint(Clean(s)) }
func Cyan(s string) string   { return cyan.Sprint(Clean(s)) }

// StrongIn is Clean(s) in bold and the given colour: one helper for what
// nesting two would escape.
func StrongIn(c color.Attribute, s string) string {
	return color.New(c, color.Bold).Sprint(Clean(s))
}
