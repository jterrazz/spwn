package ui

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// buildProgressWriter scans a Docker build stream for `Step N/M :`
// Lines and emits one tree line per step into the active Stepper.
// Every other docker stream byte is dropped on the floor — terminal
// Scrollback preserves the tree itself, no raw Dockerfile noise.
//
// Docker build emits one stream message per instruction, formatted
// As `Step N/M : RUN apt-get install ...\n` at the start of each
// Step. We summarise the instruction ("Installing packages" /
// "Installing <pkg>" / "Copying files" / etc.) so the tree stays
// Compact and readable.
//
// UX: while a step is in progress, the writer leaves a live spinner
// On its line via Stepper.TreeSpin. When the next step arrives (or
// The phase summary is printed via CompleteCurrent), that spinner
// Line is overwritten with a frozen "├ [N/M] <action> (Xs)" so the
// Terminal accumulates the full history.
type buildProgressWriter struct {
	stepper *Stepper
	buf     bytes.Buffer

	// Current live step state — set when we see `Step N/M :`, cleared
	// When we freeze the frozen tree line (either at the next Step or
	// When CompleteCurrent is called at end-of-build).
	haveCurrent bool
	curNum      string
	curTotal    string
	curAction   string
	startedAt   time.Time
}

var stepLineRE = regexp.MustCompile(`^Step (\d+)/(\d+) : (.+)$`)

func (w *buildProgressWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// Partial line — put it back and wait for more input.
			w.buf.WriteString(line)
			break
		}
		w.handleLine(strings.TrimRight(line, "\r\n"))
	}
	return len(p), nil
}

func (w *buildProgressWriter) handleLine(line string) {
	m := stepLineRE.FindStringSubmatch(line)
	if m == nil {
		return
	}
	cur, total, instruction := m[1], m[2], m[3]
	action := summariseInstruction(instruction)

	// Freeze the previous step as a permanent tree line so scrollback
	// Keeps the history (this replaces the live spinner for the
	// Previous step). Then start spinning the new one.
	w.freezeCurrent()
	w.haveCurrent = true
	w.curNum = cur
	w.curTotal = total
	w.curAction = action
	w.startedAt = time.Now()
	w.stepper.TreeSpin(formatStepText(cur, total, action, 0))
}

// freezeCurrent converts the in-progress step's spinner line into a
// Permanent tree line with the elapsed time appended. No-op when
// There is no step currently live (first call of the build, or
// After CompleteCurrent has already frozen it).
func (w *buildProgressWriter) freezeCurrent() {
	if !w.haveCurrent {
		return
	}
	elapsed := time.Since(w.startedAt)
	text, detail := formatStepTextParts(w.curNum, w.curTotal, w.curAction, elapsed)
	w.stepper.TreeLine(text, detail)
	w.haveCurrent = false
}

// CompleteCurrent flushes any live in-progress step as a frozen tree
// Line. Call this once the build finishes (either the image_built /
// Image_cached event fires, or the phase errors out) so the spinner
// Doesn't hang around above the summary line.
func (w *buildProgressWriter) CompleteCurrent() {
	w.freezeCurrent()
}

// formatStepText returns the single-line label used for the live
// Spinner; it matches the frozen tree-line shape sans the elapsed
// Detail which is meaningless while the step is still running.
func formatStepText(cur, total, action string, elapsed time.Duration) string {
	text, detail := formatStepTextParts(cur, total, action, elapsed)
	if detail == "" {
		return text
	}
	return text + " " + faint(detail)
}

// formatStepTextParts splits the step line into the (coloured) main
// Text and the trailing faint detail. Exposed to the Stepper so the
// TreeLine caller can apply its own formatting/colour to each half.
func formatStepTextParts(cur, total, action string, elapsed time.Duration) (string, string) {
	// [N/M] rendered in subtle green-cyan so the user can eyeball
	// Progress; the action body stays faint so the whole line reads
	// As "background chatter" the eye skims past.
	prefix := faint("[" + cur + "/" + total + "]")
	body := faint(action)
	text := prefix + " " + body
	if elapsed > 0 {
		return text, fmt.Sprintf("(%.1fs)", elapsed.Seconds())
	}
	return text, ""
}

// summariseInstruction turns a Dockerfile instruction into a human
// action label. Keeps the spinner readable - the full instruction
// is too noisy ("RUN apt-get update && apt-get install -y bash sh
// ls cat cp mv rm mkdir rmdir chmod chown grep sed awk...") and
// changes every frame.
func summariseInstruction(instruction string) string {
	upper := strings.ToUpper(instruction)
	switch {
	case strings.HasPrefix(upper, "FROM "):
		return "Pulling base image"
	case strings.HasPrefix(upper, "RUN APT-GET UPDATE") ||
		strings.HasPrefix(upper, "RUN APT-GET INSTALL") ||
		strings.Contains(upper, "APT-GET INSTALL"):
		return "Installing system packages"
	case strings.HasPrefix(upper, "RUN NPM INSTALL"):
		// "RUN npm install -g @foo/bar ..." -> "Installing @foo/bar"
		rest := strings.TrimPrefix(instruction, "RUN npm install")
		rest = strings.TrimPrefix(rest, " -g")
		rest = strings.TrimSpace(rest)
		if pkg := firstWord(rest); pkg != "" {
			return "Installing " + pkg
		}
		return "Installing npm package"
	case strings.HasPrefix(upper, "COPY "):
		return "Copying files"
	case strings.HasPrefix(upper, "RUN USERADD") ||
		strings.HasPrefix(upper, "RUN CHOWN") ||
		strings.HasPrefix(upper, "USER "):
		return "Setting up user"
	case strings.HasPrefix(upper, "RUN MKDIR"):
		return "Creating directories"
	case strings.HasPrefix(upper, "LABEL "):
		return "Labelling image"
	case strings.HasPrefix(upper, "ENV "):
		return "Setting environment"
	case strings.HasPrefix(upper, "WORKDIR "):
		return "Setting workdir"
	case strings.HasPrefix(upper, "VOLUME ") || strings.HasPrefix(upper, "ENTRYPOINT "):
		return "Finalising"
	default:
		return "Running step"
	}
}

func firstWord(s string) string {
	for i, r := range s {
		if r == ' ' || r == '\t' {
			return s[:i]
		}
	}
	return s
}
