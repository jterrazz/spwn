package kit

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	aliasNow        = "station.alias.now"
	aliasShell      = "station.alias.shell"
	aliasArgs       = "station.alias.args"
	aliasPositional = "station.alias.positional"
)

func AliasPositional(f *pflag.FlagSet, name string) {
	// value: an old flag whose value the new command takes as its argument
	// (power awake --for 2h is awake 2h): the notice says its value alone,
	// its default when it was not given.
	_ = f.SetAnnotation(name, aliasPositional, []string{"yes"})
}

func Alias(old string, cmd *cobra.Command, now string) *cobra.Command {
	// value: cmd as the hidden old word of a command for one release (SPEC
	// §4.3): it runs under the old name, help and completion never list it,
	// and each run names the new command on stderr.
	_, rest, _ := strings.Cut(cmd.Use, " ")
	cmd.Use = strings.TrimSpace(old + " " + rest)
	cmd.Hidden = true
	cmd.GroupID = ""
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[aliasNow] = now
	return cmd
}

func AliasArgs(cmd *cobra.Command, now string) *cobra.Command {
	// value: the arguments the new command takes in place of the old run's,
	// when the old ones do not carry over (many v2 names, one account now).
	cmd.Annotations[aliasArgs] = now
	return cmd
}

func AliasShell(cmd *cobra.Command, now string) *cobra.Command {
	// value: an old command whose new form is no station command but a
	// line of the owner's shell (run git wip is git wip): the notice says
	// that line, then the run's arguments.
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[aliasShell] = now
	return cmd
}

// invariant: the notice is the run's own words with every old one replaced,
// its arguments then its flags, a value flag as --name=value; --json and --page keep stderr empty for the script.
func notice(env Env, c *cobra.Command, args []string) {
	if line, ok := c.Annotations[aliasShell]; ok {
		env.Out.Now(strings.Join(append([]string{line}, args...), " "))
		return
	}
	var words []string
	aliased := false
	if now, ok := c.Annotations[aliasArgs]; ok {
		args = []string{now}
	}
	for p := c; p != nil && p.HasParent(); p = p.Parent() {
		word := p.Name()
		if now, ok := p.Annotations[aliasNow]; ok {
			word, aliased = now, true
		}
		if word != "" {
			words = append([]string{word}, words...)
		}
	}
	if !aliased {
		return
	}
	for _, machine := range []string{"json", "page"} {
		if f := c.Flags().Lookup(machine); f != nil && f.Changed {
			return
		}
	}
	words = append(words, args...)
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if _, ok := f.Annotations[aliasPositional]; ok {
			words = append(words, f.Value.String())
		}
	})
	c.Flags().Visit(func(f *pflag.Flag) {
		if _, ok := f.Annotations[aliasPositional]; ok {
			return
		}
		if f.Value.Type() == "bool" {
			words = append(words, "--"+f.Name)
			return
		}
		if many, ok := f.Value.(pflag.SliceValue); ok {
			for _, v := range many.GetSlice() {
				words = append(words, "--"+f.Name+"="+v)
			}
			return
		}
		words = append(words, "--"+f.Name+"="+f.Value.String())
	})
	env.Out.Now("station " + strings.Join(words, " "))
}

func Aliased(env Env, pre func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if err := pre(c, args); err != nil {
			return err
		}
		notice(env, c, args)
		return nil
	}
}
