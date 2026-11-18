package machines

type SSHTarget struct {
	// value: the machine's login and its short name, which the owner's client
	// blocks make reach it, a VM through its Mac's pipe.
	User, Address string
	// value: the command line that reaches it.
	Command string
}
