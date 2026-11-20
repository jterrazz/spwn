package setup

const defaultName = "Jean-Baptiste Terrazzoni"

type Identity struct {
	// value: who the owner is to the tools station sets up (git, ssh keys);
	// an empty name asks the default, an empty email stays unset.
	UserEmail, UserName string
}

func (i Identity) Email() string { return i.UserEmail }

func (i Identity) Name() string {
	if i.UserName != "" {
		return i.UserName
	}
	return defaultName
}
