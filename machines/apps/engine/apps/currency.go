package apps

type Currency struct {
	Verdict Verdict
	// value: the version a newer release would bring, when the method says it.
	Newest string
}

type Verdict string

const (
	Current Verdict = "current"
	Behind  Verdict = "behind"
	Unknown Verdict = "unknown"
)

func Unread(keys []string) map[string]Currency {
	// invariant: a read that did not answer leaves every key's currency
	// unknown, never behind.
	out := make(map[string]Currency, len(keys))
	for _, k := range keys {
		out[k] = Currency{Verdict: Unknown}
	}
	return out
}

func Verdicts(keys []string, behind map[string]string) map[string]Currency {
	// value: each key current, or behind at the newest version its manager
	// named for it.
	out := make(map[string]Currency, len(keys))
	for _, k := range keys {
		out[k] = Currency{Verdict: Current}
		if newest, ok := behind[k]; ok {
			out[k] = Currency{Verdict: Behind, Newest: newest}
		}
	}
	return out
}
