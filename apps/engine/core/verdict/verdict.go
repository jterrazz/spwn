package verdict

type Tone string

const (
	OK    Tone = "ok"
	Warn  Tone = "warn"
	Bad   Tone = "bad"
	Plain Tone = "plain"
	New   Tone = "new"
)

func (Tone) Enum() []string {
	return []string{string(OK), string(Warn), string(Bad), string(Plain), string(New)}
}

type Verdict struct {
	Tone     Tone
	Sentence string
}
