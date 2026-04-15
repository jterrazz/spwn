package ui

import (
	"bytes"
	"regexp"
	"strings"
)

// buildProgressWriter scans a Docker build stream for `Step N/M :`
// lines and updates the active stepper label as each step starts.
// Every other line (actual build log output) is dropped on the
// floor - the spinner is the whole UX; we don't want to spam the
// terminal with raw docker stream chatter.
//
// Docker build emits one stream message per instruction, formatted
// as `Step N/M : RUN apt-get install ...\n` at the start of each
// step. We summarise the instruction ("Installing packages" /
// "Installing <pkg>" / "Copying files" / etc.) so the label stays
// compact and readable rather than echoing the raw Dockerfile.
type buildProgressWriter struct {
	stepper *Stepper
	base    string
	buf     bytes.Buffer
}

var stepLineRE = regexp.MustCompile(`^Step (\d+)/(\d+) : (.+)$`)

func (w *buildProgressWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// Partial line - put it back and wait for more input.
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
	w.stepper.UpdateLabel(
		w.base + " " + faint("["+cur+"/"+total+"] "+action),
	)
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
