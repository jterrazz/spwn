package ui

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type buildProgressWriter struct {
	stepper *Stepper
	buf     bytes.Buffer

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

	w.freezeCurrent()
	w.haveCurrent = true
	w.curNum = cur
	w.curTotal = total
	w.curAction = action
	w.startedAt = time.Now()
	w.stepper.TreeSpin(formatStepText(cur, total, action, 0))
}

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

func formatStepText(cur, total, action string, elapsed time.Duration) string {
	text, detail := formatStepTextParts(cur, total, action, elapsed)
	if detail == "" {
		return text
	}
	return text + " " + faint(detail)
}

func formatStepTextParts(cur, total, action string, elapsed time.Duration) (string, string) {
	prefix := faint("[" + cur + "/" + total + "]")
	body := faint(action)
	text := prefix + " " + body
	if elapsed > 0 {
		return text, fmt.Sprintf("(%.1fs)", elapsed.Seconds())
	}
	return text, ""
}

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
