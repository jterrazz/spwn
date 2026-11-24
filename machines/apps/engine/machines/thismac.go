package machines

func ThisMacName(preference string, defaultName func() string) string {
	// invariant: this Mac has one name on every call: the one it was
	// registered under, else the name it takes by default, read only then.
	if preference != "" {
		return preference
	}
	return defaultName()
}
