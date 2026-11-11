package kit

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/output"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/surface/cli/words"
)

type Asker interface {
	Confirm(c page.Confirm) (bool, error)
	Form(f page.Form) (map[string]string, error)
}

type Env struct {
	Out         *output.Writer
	Prompt      Asker
	Interactive bool
	On          *On
	Use         Uses
	Stdin       io.Reader
	Stdout      io.Writer
	// value: where a leaf's call runs, and this machine; Target names the
	// machine a leaf's calls go to.
	Calls call.Router
	Self  machine.Ref
	// value: what this machine is, and the clock a document is dated by.
	Kind machine.Kind
	Now  func() time.Time
	// value: a name as this Mac's records hold it: a Mac by its full name,
	// a VM of this Mac with On set; false for any other name.
	Locate func(name string) (machine.Ref, bool)
	// value: installs one catalogue item on the target and prints its
	// outcome, as apps install does; how a terminal answers a missing need.
	Install func(ctx context.Context, item string, o IO) error
}

type IO struct {
	JSON          bool
	PageJSON      bool
	Yes           bool
	Interactive   bool
	Prompt        Asker
	ZshCompletion func(w io.Writer) error
	out           *output.Writer
	in            io.Reader
	raw           io.Writer
	ask           func() error
	values        func() (map[string]string, error)
	// value: the machine a far call ran on; its output starts with `On <it>:`.
	on       string
	envelope func() *output.Envelope
}

func (o IO) Confirmed(do func(answer string) error) error {
	// invariant: the call runs with --yes's answer, else unanswered; only
	// when its use case still asks is the question put (--yes, the
	// terminal, else a refusal), then the call runs once more with the answer.
	first := ""
	if o.Yes {
		first = confirm.Yes
	}
	err := do(first)
	answer, ok := confirm.Answer(err)
	if !ok || o.ask == nil {
		return err
	}
	if err := o.ask(); err != nil {
		return err
	}
	return do(answer)
}

func (o IO) YesAnswer() string {
	// value: what --yes answers a call before it asks: yes, or nothing.
	if o.Yes {
		return confirm.Yes
	}
	return ""
}

func (o IO) Streams() (io.Reader, io.Writer) { return o.in, o.raw }

func (o IO) Values() (map[string]string, error) {
	// value: the answers handed on stdin by their input's name; nil when the
	// command was not given any, so it asks them itself or refuses.
	if o.values == nil {
		return nil, nil
	}
	return o.values()
}

func (o IO) CanLive() bool {
	// invariant: a read goes live only on a terminal, without --json and without
	// NO_COLOR; everywhere else it prints one complete document.
	return !o.JSON && !o.PageJSON && o.out.CanLive()
}

func (o IO) Page(p page.Page, contract any) error {
	return o.out.Page(o.onLine(p), contract, o.want(output.Want{}))
}

func (o IO) PagePiped(p page.Page, contract any, piped string) error {
	return o.out.Page(o.onLine(p), contract, o.want(output.Want{Piped: []byte(piped)}))
}

func (o IO) PageRecords(p, records page.Page, contract any) error {
	return o.out.Page(o.onLine(p), contract, o.want(output.Want{Records: &records}))
}

func (o IO) want(w output.Want) output.Want {
	w.JSON, w.Page = o.JSON, o.PageJSON
	if o.JSON && o.envelope != nil {
		w.Envelope = o.envelope()
	}
	return w
}

func (o IO) LivePage(pages <-chan page.Page) error {
	if o.on == "" {
		return o.out.LivePage(pages)
	}
	named := make(chan page.Page)
	go func() {
		defer close(named)
		for p := range pages {
			named <- o.onLine(p)
		}
	}()
	return o.out.LivePage(named)
}

// invariant: SPEC §8.1: every output for another machine starts with
// `On <that machine>:`, the machine --on named by the name the interface
// gives it (SPEC §2.5: `mac-mini` for `jterrazz-mac-mini`), in place of a
// page's own.
func (o IO) onLine(p page.Page) page.Page {
	if o.on == "" {
		return p
	}
	sections := p.Sections
	if len(sections) > 0 && sections[0].ID == page.OnSection {
		sections = sections[1:]
	}
	p.Sections = append([]page.Section{page.OnMachine(o.on)}, sections...)
	return p
}

func (o IO) Raw(p []byte) error              { return o.out.Raw(p) }
func (o IO) Note(text string)                { o.out.Note(text) }
func (o IO) Wait(label string) (stop func()) { return o.out.Wait(label) }

type Leaf[I any] struct {
	Use, Short, Example string
	Group               string
	Plumbing            bool
	Args                cobra.PositionalArgs
	ValidArgs           []string
	RawArgs             bool
	Declare             func(f *pflag.FlagSet, in *I)
	Bind                func(in *I, args []string) error
	Validate            func(ctx context.Context, in *I) error
	Read                func(in *I) bool
	// value: a leaf whose call may ask a confirmation: it takes --yes, and
	// its question is put only when the use case asks it (io.Confirmed).
	Asks    bool
	Confirm func(in *I) string
	Verb    func(in *I) string
	Ask     func(in *I) (bool, error)
	// value: the refusal off a terminal without --yes, when it names what the
	// call would do; else the gate's own words.
	Refuse   func(in *I) error
	Complete func(args []string, toComplete string) []string
	// value: the refusal of a word that names an entry of the family, the
	// verb forgotten (SPEC §4.2); nil for any other word.
	Entry func(word string) error
	Run   func(ctx context.Context, in *I, io IO) error
	// value: a leaf that draws its reading as it fills needs no spinner.
	Live bool
	// value: a leaf whose answer is a page also prints it as station.page/v1.
	Pages bool
	// value: a leaf whose form the TUI fills: with --values-on-stdin, the
	// answers are read on stdin (io.Values), never in its words.
	Values bool
}

const ValuesOnStdin = "values-on-stdin"

func Command[I any](l Leaf[I], env Env) *cobra.Command {
	in := new(I)
	var wantJSON, wantPage, yes bool
	args := l.Args
	if args == nil {
		args = cobra.NoArgs
	}
	cmd := &cobra.Command{
		Use:                l.Use,
		Short:              l.Short,
		Example:            l.Example,
		GroupID:            l.Group,
		ValidArgs:          l.ValidArgs,
		DisableFlagParsing: l.RawArgs,
		Hidden:             l.Plumbing,
		Annotations:        map[string]string{owned: "leaf"},
	}
	cmd.Args = func(c *cobra.Command, a []string) error {
		if err := args(c, a); err != nil {
			if l.Entry != nil && len(a) > 0 {
				if refused := l.Entry(a[0]); refused != nil {
					return refused
				}
			}
			if what := wanted(c.Use); what != "" && len(a) == 0 {
				return fail.New(fail.Refused, words.MissingArgument, c.CommandPath(), map[string]string{"what": what})
			}
			return usage(err.Error(), helpHint(c))
		}
		return nil
	}
	if l.Complete != nil {
		cmd.ValidArgsFunction = func(_ *cobra.Command, a []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return l.Complete(a, toComplete), cobra.ShellCompDirectiveNoFileComp
		}
	}
	if l.Declare != nil {
		l.Declare(cmd.Flags(), in)
	}
	if l.Read != nil {
		cmd.Flags().BoolVar(&wantJSON, "json", false, "print the answer as JSON")
	}
	if l.Pages {
		cmd.Flags().BoolVar(&wantPage, "page", false, "print the page as station.page/v1")
		_ = cmd.Flags().MarkHidden("page")
	}
	if l.Asks {
		cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do it without asking")
	}
	var onStdin bool
	if l.Values {
		cmd.Flags().BoolVar(&onStdin, ValuesOnStdin, false, "read the form's answers on stdin, as JSON")
		_ = cmd.Flags().MarkHidden(ValuesOnStdin)
	}
	cmd.RunE = func(c *cobra.Command, a []string) error {
		if l.RawArgs && len(a) > 0 && (a[0] == "-h" || a[0] == "--help") {
			return c.Help()
		}
		o := IO{
			JSON:          wantJSON,
			PageJSON:      wantPage,
			Yes:           yes,
			Interactive:   env.Interactive,
			Prompt:        env.Prompt,
			ZshCompletion: c.Root().GenZshCompletion,
			out:           env.Out,
			envelope:      func() *output.Envelope { return env.envelope(c.Context()) },
		}
		if env.Far() {
			o.on = page.MacName(env.Target().Name, env.Self.Name)
		}
		if l.Plumbing {
			o.in, o.raw = env.Stdin, env.Stdout
		}
		if onStdin {
			o.values = func() (map[string]string, error) { return valuesOn(env.Stdin) }
		}
		err := run(c.Context(), c, l, in, a, env, o)
		err = env.answerNeed(c.Context(), err, o, func() error { return proceed(c.Context(), c, l, in, a, env, o) })
		read := err != nil && l.Read != nil && l.Read(in)
		silent := read && env.Far() && unanswered(err)
		if silent && (o.JSON || o.PageJSON) {
			// invariant: SPEC §11.1, §12.2: a read of a machine that does not
			// answer prints station.error/v1 and its block, and still exits 0,
			// as the same read does on a terminal.
			worded := Translate(err, path(c))
			env.Out.FailureJSON(worded)
			env.Out.Failure(worded)
			return nil
		}
		if read && !o.JSON && !o.PageJSON && vmStopped(err) {
			return env.Out.Finding(page.Absent, env.Target().Name+" is stopped", "start it to read its "+strings.Fields(path(c))[0])
		}
		if silent {
			// invariant: SPEC §8.1: a read of a machine that does not answer
			// prints its finding and exits 0; an action exits 1.
			return env.Out.Finding(page.Broken, append([]string{Translate(err, path(c)).Message}, lastAnswer(c.Context(), env, err)...)...)
		}
		if err != nil {
			return Translate(err, path(c))
		}
		if env.Far() {
			env.On.refresh(c.Context(), path(c))
		}
		return nil
	}
	return cmd
}

func run[I any](ctx context.Context, c *cobra.Command, l Leaf[I], in *I, args []string, env Env, o IO) error {
	if l.Bind != nil {
		if err := l.Bind(in, args); err != nil {
			return err
		}
	}
	return proceed(ctx, c, l, in, args, env, o)
}

func proceed[I any](ctx context.Context, c *cobra.Command, l Leaf[I], in *I, args []string, env Env, o IO) error {
	// value: a leaf's run past its binding, which says what it says once:
	// what a terminal runs again once a missing need is installed.
	if l.Validate != nil && !env.Far() {
		// invariant: Validate checks against this machine; a call sent
		// elsewhere is checked by its own use case, there.
		if err := l.Validate(ctx, in); err != nil {
			return err
		}
	}
	if l.Asks {
		ask := func() (bool, error) {
			// invariant: the question is put together only when it is put:
			// one that reads a machine reads nothing for a refusal.
			question := c.Name() + "?"
			if l.Confirm != nil {
				question = l.Confirm(in)
			}
			return o.Prompt.Confirm(page.Confirm{Question: question})
		}
		if l.Ask != nil {
			ask = func() (bool, error) { return l.Ask(in) }
		}
		verb := c.Name()
		if l.Verb != nil {
			verb = l.Verb(in)
		}
		kept := strings.Join(args, ", ")
		if kept == "" {
			kept = c.Name()
		}
		o.ask = func() error {
			if l.Refuse != nil && !o.Yes && !o.Interactive {
				return l.Refuse(in)
			}
			return gate(verb, kept, ask, o)
		}
	}
	if l.Read != nil && l.Read(in) && !o.JSON && !o.PageJSON && (!l.Live || !o.CanLive()) {
		stop := env.Out.Wait("reading")
		defer stop()
	}
	return l.Run(ctx, in, o)
}

func gate(verb, kept string, ask func() (bool, error), o IO) error {
	switch {
	case o.Yes:
		return nil
	case o.Interactive:
		ok, err := ask()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return fail.Interruption()
			}
			return failed(err.Error(), "re-run with --yes to skip the question", err)
		}
		if !ok {
			return fail.New(fail.Declined, confirm.Declined, kept, nil)
		}
		return nil
	default:
		return usage("refusing to "+verb+" without --yes", "re-run with --yes")
	}
}

func Group(use, short, groupID string, env Env, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:         use,
		Short:       short,
		GroupID:     groupID,
		Args:        cobra.ArbitraryArgs,
		Annotations: map[string]string{owned: "group"},
	}
	cmd.AddCommand(children...)
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return Menu(env, c)
		}
		hint := helpHint(c)
		if s := c.SuggestionsFor(args[0]); len(s) > 0 {
			hint += didYouMean(s)
		}
		return usage("unknown command \""+args[0]+"\" for \""+c.CommandPath()+"\"", hint)
	}
	return cmd
}

func Entry(verb, row string, names func() []string) func(string) error {
	return func(word string) error {
		if !slices.Contains(names(), word) {
			return nil
		}
		return fail.New(fail.Refused, words.EntryWithoutVerb, word, map[string]string{"row": row, "command": verb})
	}
}

// value: what a leaf's first argument names, from its usage line
// (`install <tool...>` wants a tool); empty for a leaf that names none.
func wanted(use string) string {
	_, after, ok := strings.Cut(use, "<")
	if !ok {
		return ""
	}
	what, _, _ := strings.Cut(after, ">")
	if strings.Contains(what, "|") {
		return ""
	}
	return strings.TrimSuffix(what, "...")
}

func Menu(env Env, cmd *cobra.Command) error {
	env.Out.Menu(func(w io.Writer) {
		cmd.SetOut(w)
		defer cmd.SetOut(nil)
		_ = cmd.Help()
	})
	return fail.Silent(fail.ExitUsage)
}

func Refuse(message, hint string) error {
	return usage(message, hint)
}

func FlagError(c *cobra.Command, err error) error {
	return usage(err.Error(), helpHint(c))
}

func helpHint(c *cobra.Command) string {
	return "run " + c.CommandPath() + " --help"
}

func path(c *cobra.Command) string {
	return strings.TrimSpace(strings.TrimPrefix(c.CommandPath(), c.Root().Name()))
}

func didYouMean(suggestions []string) string {
	return " (did you mean \"" + strings.Join(suggestions, "\" or \"") + "\"?)"
}

func Updates[T any](updates <-chan call.Update[T], build func(T) page.Page) <-chan page.Page {
	// invariant: an update in error draws nothing; the stream is still read
	// to its end, so its producer never blocks.
	out := make(chan page.Page)
	go func() {
		defer close(out)
		for u := range updates {
			if u.Err == nil {
				out <- build(u.Value)
			}
		}
	}()
	return out
}

type Spinner struct {
	io   IO
	stop func()
}

func Steps(o IO) *Spinner { return &Spinner{io: o, stop: func() {}} }

func (s *Spinner) Step(task, detail string) {
	// value: a call's Progress on the CLI: each step's label is the one
	// spinner line, replacing the last; a step that asks the owner is a line
	// of its own instead, under which its prompt waits.
	s.stop()
	why := strings.Split(detail, "\n")
	if detail == "" {
		why = nil
	}
	if len(why) > 0 && why[0] == call.Asks {
		s.stop = func() {}
		s.io.out.Ahead(append([]string{task}, words.Reasons(why[1:])...)...)
		return
	}
	s.stop = s.io.Wait(output.Said(append([]string{task}, words.Reasons(why)...)...))
}

func (*Spinner) Done(string, error) {}

func (s *Spinner) Stop() { s.stop() }
