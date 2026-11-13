package shellcmd

import (
	"bytes"
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func Command(env kit.Env) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:       "shell <zsh|prompt>",
		Short:     "The script the ~/.zshrc block evaluates, and its prompt segment",
		Args:      cobra.ArbitraryArgs,
		ValidArgs: []string{"zsh", "prompt"},
		Bind:      bind,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.Shell == prompt {
				return io.Raw([]byte(segment(env)))
			}
			var completion bytes.Buffer
			if err := io.ZshCompletion(&completion); err != nil {
				return err
			}
			script, err := call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Zsh, settingscalls.ZshIn{Completion: completion.String()}, nil)
			if err != nil {
				return err
			}
			return io.Raw([]byte(script))
		},
	}, env)
	cmd.Hidden = true
	return cmd
}

// invariant: the segment reads the choice and never marks it used: a prompt
// drawn is no command, so a terminal left alone still forgets it (SPEC §8.2).
func segment(env kit.Env) string {
	if env.Use == nil {
		return ""
	}
	c, ok := env.Use.Read()
	if !ok || c.Machine == "" || !c.Gone.IsZero() {
		return ""
	}
	return "on " + c.Machine + "\n"
}
