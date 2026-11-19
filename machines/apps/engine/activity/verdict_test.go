package activity_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

func TestVerdictOfAReport(t *testing.T) {
	broken := finding.Finding{Severity: finding.Broken, Subject: "disk", Text: "Disk almost full: 4 % free"}
	look := finding.Finding{Severity: finding.NeedsALook, Subject: "memory", Text: "Memory pressure"}
	for _, tc := range []struct {
		name string
		r    activity.Report
		want verdict.Verdict
	}{
		{"kept and calm", activity.Report{Since: start}, verdict.Verdict{Tone: verdict.OK, Sentence: activity.SentenceCalm, Line: activity.LineKept}},
		{"read now and calm", activity.Report{}, verdict.Verdict{Tone: verdict.OK, Sentence: activity.SentenceCalm}},
		{"the first finding leads", activity.Report{Since: start, Findings: []finding.Finding{broken, look}}, verdict.Verdict{Tone: verdict.Bad, Sentence: broken.Text, Line: "disk"}},
		{"a need of a look is a warning", activity.Report{Findings: []finding.Finding{look}}, verdict.Verdict{Tone: verdict.Warn, Sentence: look.Text, Line: "memory"}},
		{"a finding with a detail says it under its sentence", activity.Report{Findings: []finding.Finding{{Severity: finding.NeedsALook, Subject: "cpu", Text: "infra busy: 96 % for 10 min", Detail: "go test holds 97 % of its 4 CPUs"}}}, verdict.Verdict{Tone: verdict.Warn, Sentence: "infra busy: 96 % for 10 min", Line: "go test holds 97 % of its 4 CPUs"}},
		{"a silent Meter is never calm", activity.Report{Silent: true}, verdict.Verdict{Tone: verdict.Warn, Sentence: activity.SentenceSilent}},
	} {
		if got := activity.VerdictOf(tc.r); got != tc.want {
			t.Errorf("%s: verdict %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
