package calls

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps"
)

func TestAnAppsAnswerLeavesWithItsToolsWordsScreened(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	out, _ := interrupted(context.Background(), []apps.Outcome{{
		Name: "node", Detail: "login --token " + token + ": refused", Reasons: []string{"asked by " + token},
		Steps: []apps.Step{{Package: "git-identity", Detail: "config --global token " + token + ": refused"}},
	}}, nil)
	if len(out) != 1 || len(out[0].Steps) != 1 || len(out[0].Reasons) != 1 {
		t.Fatalf("the outcome lost its parts: %+v", out)
	}
	o := out[0]
	for _, words := range []string{o.Detail, o.Reasons[0], o.Steps[0].Detail} {
		if strings.Contains(words, token) || !strings.Contains(words, "withheld") {
			t.Errorf("an answer said %q", words)
		}
	}
}
