package ports

type Answer string // value: whether a port's machine answers on it now, whoever may reach it.

const (
	Answers     Answer = ""
	Unreachable Answer = "unreachable"
	Starting    Answer = "starting"
)

type Halt int // value: why a VM whose ports were heard does not run them now.

const (
	Runs Halt = iota
	Down
	Restarting
	StoppedOnPurpose
)

func AnswerOf(h Halt) (Answer, bool) {
	// invariant: a port follows its VM: unreachable while it is down,
	// starting while it boots again; a VM stopped on purpose lists none, as
	// nothing is meant to answer there.
	switch h {
	case Down:
		return Unreachable, true
	case Restarting:
		return Starting, true
	case StoppedOnPurpose:
		return Answers, false
	}
	return Answers, true
}
