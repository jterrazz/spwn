package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/output"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func TestAFindingWithholdsTheTokenOfTheErrorItSays(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	var out bytes.Buffer
	w := output.NewWriter(output.Stream{W: &out}, output.Stream{W: &bytes.Buffer{}}, nil)
	if err := w.Finding(page.Broken, "fetch https://bot:"+token+"@host/x refused", "last answer "+token); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); strings.Contains(got, token) || !strings.Contains(got, "https://withheld@host/x refused") {
		t.Errorf("the finding said %q", got)
	}
}
