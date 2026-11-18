package help

import (
	"image/color"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"

	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func FangScheme(pick lipgloss.LightDarkFunc) fang.ColorScheme {
	light, dark := token.Colors(false), token.Colors(true)
	role := func(of func(token.Palette) color.Color) color.Color { return pick(of(light), of(dark)) }
	text := role(func(p token.Palette) color.Color { return p.Text })
	muted := role(func(p token.Palette) color.Color { return p.Muted })
	accent := role(func(p token.Palette) color.Color { return p.Accent })
	bad := role(func(p token.Palette) color.Color { return p.Bad })
	surface := role(func(p token.Palette) color.Color { return p.Surface })
	return fang.ColorScheme{
		Base:           text,
		Title:          accent,
		Description:    text,
		Codeblock:      surface,
		Program:        accent,
		DimmedArgument: muted,
		Comment:        muted,
		Flag:           accent,
		FlagDefault:    muted,
		Command:        accent,
		QuotedString:   text,
		Argument:       text,
		Help:           muted,
		Dash:           muted,
		ErrorHeader:    [2]color.Color{surface, bad},
		ErrorDetails:   bad,
	}
}
