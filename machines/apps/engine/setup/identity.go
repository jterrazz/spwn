package setup

type Identity struct {
	// value: who the owner is to the tools station sets up (git, ssh keys);
	// an empty value stays unset.
	UserEmail, UserName string
}

func (i Identity) Email() string { return i.UserEmail }

func (i Identity) Name() string { return i.UserName }
