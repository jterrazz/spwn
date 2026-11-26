package kit

import (
	"context"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

func Screened(err error, path string) *fail.Error {
	// invariant: an error worded for a surface that draws its words itself,
	// outside page.Error and the error block, leaves with its message
	// screened.
	worded := Translate(err, path)
	if worded == nil {
		return nil
	}
	said := *worded
	said.Message = secret.Kept(said.Message)
	return &said
}

func Translate(err error, path string) *fail.Error {
	if err == nil {
		return nil
	}
	hint := "run " + strings.TrimSpace("station "+path) + " --help"
	var worded *fail.Error
	if errors.As(err, &worded) {
		if word, ok := words.Codes()[worded.Code]; ok && worded.Message == "" {
			worded.Message, worded.Hint = word(worded)
		}
		var said *fail.Said
		if worded.Message == "" && errors.As(worded.Cause, &said) {
			worded.Message, worded.Hint = said.Message, said.Hint
		}
		if worded.Message != "" && worded.Hint == "" && worded.Type != fail.Declined {
			worded.Hint = hint
		}
		return worded
	}
	for _, failure := range []func(error) (*fail.Error, bool){
		appsFailure,
		settingsFailure,
		setupFailure,
		machinesFailure,
		imagesFailure,
		tailnetFailure,
		awakeFailure,
		engineFailure,
		saidFailure,
		preferencesFailure,
	} {
		if f, ok := failure(err); ok {
			// invariant: every worded error carries its one repair; a family's
			// own that names none gets this command's help.
			if f.Hint == "" {
				f.Hint = hint
			}
			return f
		}
	}
	if errors.Is(err, context.Canceled) {
		return fail.Interruption()
	}
	message := err.Error()
	if strings.HasPrefix(message, "unknown command ") {
		return unknownCommand(message, path, hint)
	}
	return failed(message, hint, err)
}

func unknownCommand(message, path, hint string) *fail.Error {
	// invariant: a word the tree does not know is answered with the words it
	// meant, at the root and under a noun alike, as commands to paste.
	first, listed, _ := strings.Cut(message, "\n\nDid you mean this?\n")
	first = strings.TrimSpace(first)
	near := strings.Fields(listed)
	if word, ok := strings.CutSuffix(first, " for \"station\""); ok {
		detail := map[string]string{"suggest": strings.Join(near, " ")}
		return Translate(fail.New(fail.Refused, words.UnknownCommand, strings.Trim(strings.TrimPrefix(word, "unknown command "), "\""), detail), path)
	}
	if len(near) > 0 {
		hint = words.Try(path, near)
	}
	return usage(first, hint)
}

func Refusal(code fail.Code, subject string) string {
	// value: the words of a refusal a result carries by its code and
	// subject, as Translate words one raised; the code itself where this
	// binary words it not.
	if worded := Translate(fail.New("", code, subject, nil), ""); worded.Message != "" {
		return worded.Message
	}
	return string(code)
}

func AdminApply(setting, mac string, thisMac bool) string {
	// value: the command that applies an admin setting on that Mac's own
	// terminal, in the words' table.
	return words.AdminApply(setting, mac, thisMac)
}

const MachinesSSHUse = words.MachinesSSHUse

func SSHConfigNotRefreshed(cause string) string { return words.SSHConfigNotRefreshed(cause) }

func Worded(outcomes []apps.Outcome) []apps.Outcome {
	// value: apps outcomes with their typed reasons worded, for a leaf's page.
	return words.Outcomes(outcomes)
}

func ApplyWords(interrupted bool) []string {
	// value: the words of a line of apply its run did not finish: the one
	// it was interrupted at, or one it did not reach.
	if interrupted {
		return []string{words.Interrupted, words.ApplyResumes}
	}
	return []string{words.ApplyNotRun, words.ApplyResumes}
}

func UpgradeWords(interrupted bool) []string {
	// value: the words of a line of upgrade its run did not finish.
	if interrupted {
		return []string{words.Interrupted, words.UpgradeResumes}
	}
	return []string{words.ApplyNotRun, words.UpgradeResumes}
}
