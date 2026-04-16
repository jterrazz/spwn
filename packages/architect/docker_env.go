package architect

import "spwn.sh/packages/auth"

// DockerEnvArgs returns -e flags for docker exec/create that inject
// every resolved AI-provider credential. Thin adapter over
// auth.ResolveAll() for the Docker shell; kept here (not in auth)
// so the auth package stays runtime-neutral.
func DockerEnvArgs() []string {
	var args []string
	for _, cred := range auth.ResolveAll() {
		if cred.Type == auth.CredTypeNone || cred.Token == "" {
			continue
		}
		args = append(args, "-e", cred.EnvVar+"="+cred.Token)
	}
	return args
}

// DockerEnvVars returns key=value env strings for container creation
// via the Docker API (as opposed to the CLI flags from DockerEnvArgs).
func DockerEnvVars() []string {
	var envs []string
	for _, cred := range auth.ResolveAll() {
		if cred.Type == auth.CredTypeNone || cred.Token == "" {
			continue
		}
		envs = append(envs, cred.EnvVar+"="+cred.Token)
	}
	return envs
}
