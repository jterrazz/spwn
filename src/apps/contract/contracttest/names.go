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
