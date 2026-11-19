package kit

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

type Choice struct {
	// value: the machine this terminal uses (SPEC §8.2), a surface
	// preference kept per STATION_TERMINAL; empty is This Mac.
	Machine  string    `json:"machine,omitempty"`
	Previous string    `json:"previous,omitempty"`
	Since    time.Time `json:"since,omitzero"`
	Used     time.Time `json:"used,omitzero"`
	// value: when this binary deleted the machine, which the next command
	// in the terminal says.
	Gone time.Time `json:"gone,omitzero"`
}

type Uses interface {
	// value: whether this terminal has the shell block (STATION_TERMINAL).
	Block() bool
	Now() time.Time
	// value: this terminal's choice; false for none, and for one unused
	// for ForgottenAfter, which the read forgets.
	Read() (Choice, bool)
	// value: keeps the choice, stamped used now; a choice of This Mac with
	// no previous machine is no choice, and is removed.
	Write(c Choice) error
	// value: marks every terminal's choice of machine as gone at at.
	Gone(machine string, at time.Time)
}

const ForgottenAfter = 7 * 24 * time.Hour

func Using(env Env, next func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	// invariant: precedence --on, then this terminal's choice, then This Mac;
	// the commands --on does not apply to run here whatever the choice.
	return func(c *cobra.Command, args []string) error {
		if err := next(c, args); err != nil {
			return err
		}
		if env.On == nil || env.Use == nil || env.On.Machine != "" || !env.Use.Block() {
			return nil
		}
		p := path(c)
		if refusesOn(p) || c.HasParent() && c.Annotations[owned] == "" {
			return nil
		}
		choice, ok := env.Use.Read()
		if !ok || choice.Machine == "" {
			return nil
		}
		if gone, ok := goneOf(c.Context(), env, choice); ok {
			env.Out.Note(choice.Machine+" is gone"+gone, "back on This Mac")
			return env.Use.Write(Choice{})
		}
		env.On.Used = choice.Machine
		return env.Use.Write(choice)
	}
}

func goneOf(ctx context.Context, env Env, c Choice) (string, bool) {
	if !c.Gone.IsZero() {
		return " (deleted at " + Clock(c.Gone, env.Use.Now()) + ")", true
	}
	if env.On.Known == nil {
		return "", false
	}
	if known, err := env.On.Known(ctx, c.Machine); err != nil || known {
		return "", false
	}
	return "", true
}

func Clock(t, now time.Time) string {
	// value: a time of this terminal's choice as the owner reads it: the
	// hour today, the day and the hour before.
	t, now = t.Local(), now.Local()
	if y, m, d := t.Date(); y == now.Year() && m == now.Month() && d == now.Day() {
		return t.Format("15:04")
	}
	return t.Format("Jan 2 15:04")
}
