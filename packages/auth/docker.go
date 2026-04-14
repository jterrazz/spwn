package auth

// DockerEnvArgs returns -e flags for docker exec/create that inject
// all resolved credentials. This replaces the 5x duplicated pattern.
func DockerEnvArgs() []string {
	var args []string
	creds := ResolveAll()
	for _, cred := range creds {
		if cred.Type == CredTypeNone || cred.Token == "" {
			continue
		}
		args = append(args, "-e", cred.EnvVar+"="+cred.Token)
	}
	return args
}

// DockerEnvVars returns key=value env strings for container creation.
func DockerEnvVars() []string {
	var envs []string
	creds := ResolveAll()
	for _, cred := range creds {
		if cred.Type == CredTypeNone || cred.Token == "" {
			continue
		}
		envs = append(envs, cred.EnvVar+"="+cred.Token)
	}
	return envs
}
