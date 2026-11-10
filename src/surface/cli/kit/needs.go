package kit

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/surface/cli/words"
)

// invariant: a call refused for a need station installs is answered on a
// terminal: the need's line, the question, the install, then the leaf once
// more; piped, as JSON or for a need station installs none, it stays the
// refusal, and answered no it ends there, the line already said.
func (e Env) answerNeed(ctx context.Context, err error, o IO, again func() error) error {
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != call.NeedMissing || refused.Detail["item"] == "" {
		return err
	}
	if !e.Interactive || e.Install == nil || e.Prompt == nil || o.JSON || o.PageJSON {
		return err
	}
	item := refused.Detail["item"]
	message, hint := words.Codes()[call.NeedMissing](refused)
	if printed := e.Out.Finding(page.Absent, message, hint); printed != nil {
		return printed
	}
	yes, asked := e.Prompt.Confirm(page.Confirm{Question: words.NeedAsk(item)})
	switch {
	case errors.Is(asked, context.Canceled):
		return fail.Interruption()
	case asked != nil:
		return failed(asked.Error(), hint, asked)
	case !yes:
		return fail.Silent(fail.ExitUsage)
	}
	if installed := e.Install(ctx, item, o); installed != nil {
		return installed
	}
	return again()
}
