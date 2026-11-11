package contracttest

func HostilePackages() []string {
	// value: the packages a hostile item or peer gives an installer, each
	// refused before an argv carries it, since a package is an argument of
	// a program run as root.
	return []string{"-oDPkg::Pre-Invoke::=id>/tmp/pwn", "-oAPT::Update::Pre-Invoke::=id", "--allow-unauthenticated", "-s", "jq curl", "../jq", ""}
}

func HostileVersions() []string {
	// value: the versions a hostile pin asks, refused as its package is.
	return []string{"1.0 -oDPkg::Pre-Invoke::=id", "-o", " 1.0"}
}

func HostileCommands() []string {
	// value: the commands a hostile item, peer or record names, each
	// refused before a path station writes or removes is built of it.
	return []string{"../../.profile", "../../.ssh/authorized_keys", "../cmd", "/etc/passwd", "a/b", "..", ".", "-x", ""}
}

func HostileRepos() []string {
	// value: the repos a hostile item, peer or record names, refused as a
	// command is, since a repo builds a route and a record's path.
	return []string{"a/../../../../../tmp/x", "../../../x", "acme/..", "acme/x/../../y", "noslash", "-a/b", ""}
}
