package engine

const (
	// value: the machine a watch of activity means when it names none: This
	// Mac, the engine's own.
	ThisMac = ""
)

type Watching struct {
	// value: the opening of an engine.watch: the machines it holds, This Mac
	// left unnamed.
	Machines []string `json:"machines"`
}
