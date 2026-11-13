package setup

const (
	defaultEmail = "admin@jterrazz.com"
	defaultName  = "Jean-Baptiste Terrazzoni"
)

type Identity struct {
	// value: who the owner is to the tools station sets up (git, ssh keys);
	// empty asks the default.
	UserEmail, UserName string
}

func (i Identity) Email() string {
	if i.UserEmail != "" {
		return i.UserEmail
	}
	return defaultEmail
}

func (i Identity) Name() string {
	if i.UserName != "" {
		return i.UserName
	}
	return defaultName
}
