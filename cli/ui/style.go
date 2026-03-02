package ui

import "github.com/fatih/color"

var (
	green = color.New(color.FgGreen)
	red   = color.New(color.FgRed)
	dim   = color.New(color.Faint)
	bold  = color.New(color.Bold)
)

func check() string  { return green.Sprint("✓") }
func cross() string  { return red.Sprint("✗") }
func faint(s string) string { return dim.Sprint(s) }
func strong(s string) string { return bold.Sprint(s) }
