package output

import (
	"io"
	"strconv"
)

type Mode int

const (
	TTY Mode = iota
	Pipe
	JSON
)

type Stream struct {
	W          io.Writer
	IsTerminal bool
	Width      int
	Height     int
	Dark       bool
}

type Context struct {
	Mode  Mode
	Width int
	Dark  bool
}

func Detect(s Stream, env map[string]string, wantJSON bool) Context {
	c := Context{Mode: Pipe, Width: 80, Dark: s.Dark}
	switch {
	case wantJSON:
		c.Mode = JSON
	case s.IsTerminal && env["NO_COLOR"] == "":
		c.Mode = TTY
	}
	if n, err := strconv.Atoi(env["COLUMNS"]); err == nil && n > 0 {
		c.Width = n
	} else if s.Width > 0 {
		c.Width = s.Width
	}
	return c
}
