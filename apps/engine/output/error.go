package output

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

const ErrorSchema = "station.error/v1"

type errorJSON struct {
	Schema  string   `json:"schema"`
	Type    string   `json:"type"`
	Code    *string  `json:"code"`
	Subject *string  `json:"subject"`
	Message string   `json:"message"`
	Hint    string   `json:"hint"`
	Fix     *fixJSON `json:"fix"`
}

type fixJSON struct {
	Call    *string `json:"call"`
	Command *string `json:"command"`
	Path    *string `json:"path"`
}

func (w *Writer) FailureJSON(err error) {
	// value: a refusal or a failure as station.error/v1 on stdout, beside the
	// block on stderr, for a run that asked --json (SPEC §11.1).
	var e *fail.Error
	if !errors.As(err, &e) {
		e = &fail.Error{Message: err.Error(), Exit: fail.ExitFailure}
	}
	if e.Message == "" {
		return
	}
	doc := errorJSON{Schema: ErrorSchema, Type: string(typeOf(e)), Code: orNull(string(e.Code)), Subject: orNull(e.Subject), Message: e.Message, Hint: e.Hint}
	if f := e.Fix; f.Call != "" || f.Command != "" || f.Path != "" {
		doc.Fix = &fixJSON{Call: orNull(f.Call), Command: orNull(f.Command), Path: orNull(f.Path)}
	}
	data, mErr := json.MarshalIndent(doc, "", "  ")
	if mErr != nil {
		return
	}
	w.settle()
	_, _ = io.WriteString(w.stdout.W, string(data)+"\n")
}

// invariant: an error worded before v3 carries only its exit code; its type
// is the one that code stands for.
func typeOf(e *fail.Error) fail.Type {
	if e.Type != "" {
		return e.Type
	}
	switch e.Exit {
	case fail.ExitUsage:
		return fail.Refused
	case fail.ExitInterrupted:
		return fail.Interrupted
	case fail.ExitOK, fail.ExitFailure:
	}
	return fail.Failed
}

func orNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
