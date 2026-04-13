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

func check() string         { return green.Sprint("✓") }
func warn() string          { return yellow.Sprint("!") }
func cross() string         { return red.Sprint("✗") }
func faint(s string) string  { return dim.Sprint(s) }
func strong(s string) string { return bold.Sprint(s) }

// Exported color helpers for use outside the ui package.
func Faint(s string) string  { return dim.Sprint(s) }
func Strong(s string) string { return bold.Sprint(s) }
func Green(s string) string  { return green.Sprint(s) }
func Yellow(s string) string { return yellow.Sprint(s) }
func Red(s string) string    { return red.Sprint(s) }
func Cyan(s string) string   { return cyan.Sprint(s) }
