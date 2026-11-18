package organism

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"charm.land/huh/v2"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/atom"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func FormStyles(isDark bool) *huh.Styles {
	st := token.New(isDark)
	s := huh.ThemeBase(isDark)
	s.Focused.Title = st.Title
	s.Focused.Description = st.Muted
	s.Focused.ErrorMessage = st.Bad
	s.Focused.ErrorIndicator = st.Bad
	s.Focused.SelectSelector = st.Accent
	s.Focused.Option = st.Text
	s.Focused.SelectedOption = st.Accent
	s.Focused.FocusedButton = st.Button
	s.Focused.BlurredButton = st.Idle
	s.Blurred.Title = st.Muted
	s.Blurred.Description = st.Subtle
	s.Blurred.FocusedButton = st.Button
	s.Blurred.BlurredButton = st.Idle
	return s
}

func ConfirmField(c page.Confirm) *huh.Confirm {
	yes, no := c.Yes, c.No
	if yes == "" {
		yes = "Yes"
	} else {
		yes = strings.ToUpper(yes[:1]) + yes[1:]
	}
	if no == "" {
		no = "No"
	}
	f := huh.NewConfirm().Title(c.Question).Affirmative(yes).Negative(no)
	if len(c.Lines) > 0 {
		lines := make([]string, 0, len(c.Lines))
		for _, l := range c.Lines {
			lines = append(lines, atom.Show(l))
		}
		f = f.Description(strings.Join(lines, "\n"))
	}
	return f
}

func Fields(f page.Form) (fields []huh.Field, read func() map[string]string) {
	fields = make([]huh.Field, 0, len(f.Fields))
	getters := make(map[string]func() string, len(f.Fields))
	for _, fd := range unfolded(f.Fields) {
		field, get := Field(fd)
		fields = append(fields, field)
		getters[fd.Key] = get
	}
	return fields, func() map[string]string {
		values := make(map[string]string, len(getters))
		for k, get := range getters {
			values[k] = get()
		}
		return values
	}
}

func unfolded(fields []page.Field) []page.Field {
	out := make([]page.Field, 0, len(fields))
	for _, f := range fields {
		if f.Kind == page.FieldDisclosure {
			out = append(out, unfolded(f.Fields)...)
			continue
		}
		out = append(out, f)
	}
	return out
}

func Field(f page.Field) (field huh.Field, read func() string) {
	value := f.Default
	switch f.Kind {
	case page.FieldSelect, page.FieldSegmented, page.FieldPick:
		if value == "" && len(f.Options) > 0 {
			value = f.Options[0].Value
		}
		opts := make([]huh.Option[string], 0, len(f.Options))
		for _, o := range f.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			if o.Note != "" {
				label += atom.Spaces(token.Gap) + o.Note
			}
			opts = append(opts, huh.NewOption(label, o.Value))
		}
		s := huh.NewSelect[string]().Title(f.Label).Options(opts...).Value(&value)
		if f.Note != "" {
			s = s.Description(f.Note)
		}
		return s, func() string { return value }
	case page.FieldMulti:
		// value: a multiselect's answer is its chosen values joined by commas,
		// its Default the ones checked at first.
		chosen := strings.Split(value, ",")
		opts := make([]huh.Option[string], 0, len(f.Options))
		for _, o := range f.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			if o.Note != "" {
				label += atom.Spaces(token.Gap) + o.Note
			}
			opts = append(opts, huh.NewOption(label, o.Value).Selected(slices.Contains(chosen, o.Value)))
		}
		picked := make([]string, 0, len(f.Options))
		for _, o := range f.Options {
			if slices.Contains(chosen, o.Value) {
				picked = append(picked, o.Value)
			}
		}
		m := huh.NewMultiSelect[string]().Title(f.Label).Value(&picked).Options(opts...).Height(len(opts) + 2)
		return m, func() string { return strings.Join(picked, ",") }
	case page.FieldConfirm:
		answer := value == strconv.FormatBool(true)
		c := huh.NewConfirm().Title(f.Label).Value(&answer)
		if f.Note != "" {
			c = c.Description(f.Note)
		}
		return c, func() string { return strconv.FormatBool(answer) }
	case page.FieldSecret:
		return input(f, &value).EchoMode(huh.EchoModePassword), func() string { return value }
	case page.FieldCommand, page.FieldNote, page.FieldRows:
		// value: what a sheet only says (its line, a hint's command, what
		// listens) is a note in the terminal, answering nothing.
		said := f.Note
		if f.Hint != nil {
			said = strings.TrimSpace(said + "\n" + f.Hint.Command)
		}
		return huh.NewNote().Title(f.Label).Description(said), func() string { return "" }
	case page.FieldText, page.FieldSlider, page.FieldDisclosure:
	}
	return input(f, &value), func() string { return value }
}

func input(f page.Field, value *string) *huh.Input {
	in := huh.NewInput().Title(f.Label).Value(value)
	if f.Note != "" {
		in = in.Description(f.Note)
	}
	if f.Required {
		in = in.Validate(func(v string) error {
			if v == "" {
				return errors.New(f.Label + " is required")
			}
			return nil
		})
	}
	return in
}
