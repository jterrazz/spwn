package ui

import "github.com/spf13/cobra"

// ExperimentalAnnotation is the cobra Annotations key set on commands
// whose surface is not yet stable. Callers can read this to filter, render,
// or warn about experimental commands.
const ExperimentalAnnotation = "spwn:experimental"

// experimentalNotice is the one-line warning appended to a command's Long
// description so `--help` shows it without requiring a custom help template.
const experimentalNotice = "\n\n⚠ Experimental: this command is in development and may change or break without notice."

// MarkExperimental tags a cobra command (and every subcommand below it)
// as experimental. The marker shows up in three places:
//
//  1. cmd.Annotations[ExperimentalAnnotation] = "true" — for programmatic
//     filtering (e.g. grouped help renderers).
//  2. cmd.Short gets a "[experimental] " prefix so list views surface it.
//  3. cmd.Long gets the experimentalNotice appended so `<cmd> --help` shows
//     it in plain prose without any template magic.
//
// Idempotent: calling it twice on the same command does not duplicate the
// marker.
func MarkExperimental(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	if cmd.Annotations[ExperimentalAnnotation] == "true" {
		return
	}
	cmd.Annotations[ExperimentalAnnotation] = "true"

	const prefix = "[experimental] "
	if cmd.Short != "" && !startsWith(cmd.Short, prefix) {
		cmd.Short = prefix + cmd.Short
	}
	if cmd.Long != "" && !contains(cmd.Long, "⚠ Experimental:") {
		cmd.Long += experimentalNotice
	} else if cmd.Long == "" && cmd.Short != "" {
		cmd.Long = cmd.Short + experimentalNotice
	}
}

// IsExperimental returns true if a command was tagged via MarkExperimental.
func IsExperimental(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Annotations == nil {
		return false
	}
	return cmd.Annotations[ExperimentalAnnotation] == "true"
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
