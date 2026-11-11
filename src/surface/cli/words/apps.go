package words

import (
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
)

const (
	// value: how an install, update or remove that was interrupted reads.
	Interrupted = "interrupted (Ctrl-C)"
)

func Outcomes(outcomes []apps.Outcome) []apps.Outcome {
	// value: the outcomes with the words of their typed reasons, where the
	// domain kept a reason and no words.
	out := make([]apps.Outcome, len(outcomes))
	for i, o := range outcomes {
		if o.Why == apps.WhyInterrupted && o.Detail == "" {
			o.Detail = Interrupted
		}
		if len(o.RequiredBy) > 0 {
			o.Reasons = []string{RequiredBy(o.RequiredBy)}
		}
		steps := make([]apps.Step, len(o.Steps))
		for j, s := range o.Steps {
			if s.Why == apps.WhyInterrupted && s.Detail == "" {
				s.Detail = Interrupted
			}
			steps[j] = s
		}
		o.Steps = steps
		out[i] = o
	}
	return out
}

func RequiredBy(names []string) string {
	// value: why an app joined a run: the apps of it that require it.
	return "required by " + strings.Join(names, ", ")
}

func Reasons(lines []string) []string {
	// value: a step's typed reasons in the owner's words, in their order.
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, apps.ReasonRequiredBy):
			out = append(out, RequiredBy(strings.Split(strings.TrimPrefix(l, apps.ReasonRequiredBy), ", ")))
		case l == apps.ReasonNeeds+string(catalog.NeedSudo):
			out = append(out, "asks your password once")
		}
	}
	return out
}
