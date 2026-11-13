package tty

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/atom"
	"github.com/jterrazz/studio-station/apps/engine/ui/molecule"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func ErrorBlock(e page.Error, colored, dark bool) string {
	block := strings.Join(molecule.ErrorLines(token.New(dark), e), "\n") + "\n"
	if !colored {
		return ansi.Strip(block)
	}
	return block
}

func NowLine(command string, colored, dark bool) string {
	// value: the one Muted line an old word prints on stderr: the command it is now.
	line := token.New(dark).Muted.Render(token.Arrow+" now: "+command) + "\n"
	if !colored {
		return ansi.Strip(line)
	}
	return line
}

func AheadLine(label string, colored, dark bool) string {
	// value: a step said before a program asks on the terminal, its `▸`
	// Muted; without color, the glyph and words alone.
	line := token.New(dark).Muted.Render(token.Cursor) + atom.Spaces(1) + label
	if !colored {
		return ansi.Strip(line)
	}
	return line
}

func FindingLine(m page.Mark, parts ...string) string {
	return atom.Glyph(m) + atom.Spaces(1) + strings.Join(parts, token.Sep) + "\n"
}

func NoteLine(parts ...string) string {
	return strings.Join(parts, token.Sep) + "\n"
}

func Wait(out io.Writer, label string, dark bool) (stop func()) {
	spin := atom.Spinner(token.New(dark))
	var mu sync.Mutex
	var ticker *time.Ticker
	drawn, stopped := false, false
	done := make(chan struct{})
	frame := 0
	frames := spin.Spinner.Frames
	draw := func() {
		mu.Lock()
		defer mu.Unlock()
		if stopped {
			return
		}
		drawn = true
		_, _ = io.WriteString(out, "\r"+spin.Style.Render(frames[frame%len(frames)]+atom.Spaces(1)+label)+ansi.EraseLineRight)
		frame++
	}
	timer := time.AfterFunc(token.WaitDelay, func() {
		draw()
		mu.Lock()
		if stopped {
			mu.Unlock()
			return
		}
		ticker = time.NewTicker(spin.Spinner.FPS)
		mu.Unlock()
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					draw()
				}
			}
		}()
	})
	return func() {
		timer.Stop()
		mu.Lock()
		defer mu.Unlock()
		if stopped {
			return
		}
		stopped = true
		close(done)
		if ticker != nil {
			ticker.Stop()
		}
		if drawn {
			_, _ = io.WriteString(out, "\r"+ansi.EraseEntireLine)
		}
	}
}
