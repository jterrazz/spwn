package atom

import (
	"charm.land/bubbles/v2/spinner"

	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func Spinner(st token.Styles) spinner.Model {
	frames := make([]string, 0, len(token.Spinner))
	for _, r := range token.Spinner {
		frames = append(frames, string(r))
	}
	return spinner.New(spinner.WithSpinner(spinner.Spinner{Frames: frames, FPS: token.SpinnerFrame}), spinner.WithStyle(st.Muted))
}
