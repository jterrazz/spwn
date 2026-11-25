package kit

import (
	"context"
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/page"
)

type spyAsker struct{ asked int }

func (s *spyAsker) Confirm(page.Confirm) (bool, error) {
	s.asked++
	return true, nil
}

func (s *spyAsker) Form(page.Form) (map[string]string, error) {
	s.asked++
	return nil, nil
}

func TestAnyRunUnderJSONIsNeverInteractiveWhateverTheTerminalSays(t *testing.T) {
	for _, flag := range []string{"", "--json", "--page"} {
		spy := &spyAsker{}
		var interactive bool
		cmd := Command(Leaf[struct{}]{
			Use: "ask", Read: func(*struct{}) bool { return false }, Pages: true,
			Run: func(_ context.Context, _ *struct{}, io IO) error {
				interactive = io.Interactive
				return nil
			},
		}, Env{Interactive: true, Prompt: spy})
		args := []string{}
		if flag != "" {
			args = append(args, flag)
		}
		cmd.SetArgs(args)
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("%q: %v", flag, err)
		}
		if want := flag == ""; interactive != want {
			t.Errorf("%q on a terminal: interactive %v, want %v", flag, interactive, want)
		}
		if spy.asked != 0 {
			t.Errorf("%q: %d questions asked, want none", flag, spy.asked)
		}
	}
}

func TestAWordTheTreeDoesNotKnowIsAnsweredWithTheWordsNearIt(t *testing.T) {
	known := []string{"machines", "apps", "chatgpt", "linux-1"}
	for _, c := range []struct {
		word string
		want []string
	}{
		{"machnies", []string{"machines"}},
		{"chatgtp", []string{"chatgpt"}},
		{"linux1", []string{"linux-1"}},
		{"qa", nil},
		{"apps", nil},
		{"APPS", nil},
		{"appz", []string{"apps"}},
	} {
		if got := Suggest(c.word, known); !slices.Equal(got, c.want) {
			t.Errorf("%q: got %v, want %v", c.word, got, c.want)
		}
	}
}

func TestAnUnknownWordUnderANounOffersTheCommandsNearIt(t *testing.T) {
	noun := &cobra.Command{Use: "machines"}
	root := &cobra.Command{Use: "station"}
	root.AddCommand(noun)
	noun.AddCommand(&cobra.Command{Use: "delete", Run: func(*cobra.Command, []string) {}})
	got := unknownCommand(UnknownCommand(noun, "delet").Error(), "machines", "run station machines --help")
	if got.Hint != "station machines delete" {
		t.Errorf("got hint %q, want the command to paste", got.Hint)
	}
	if got := unknownCommand(UnknownCommand(noun, "zzzz").Error(), "machines", "run station machines --help"); got.Hint != "run station machines --help" {
		t.Errorf("got hint %q, want the help when nothing is near", got.Hint)
	}
}
