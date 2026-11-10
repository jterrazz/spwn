package words

import "github.com/jterrazz/studio-station/src/apps"

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
