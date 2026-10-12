package architect

import (
	"os"

	"spwn.sh/packages/authentication"
)

var toolPassthroughEnvVars = []string{}

// DockerEnvArgs returns -e flags for docker exec/create that inject
// Every resolved AI-provider credential plus any tool-passthrough
// Env vars present on the host. Thin adapter over authentication.ResolveAll()
// For the Docker shell; kept here (not in auth) so the auth package
// Stays runtime-neutral.
func DockerEnvArgs() []string {
	var args []string
	for _, cred := range authentication.ResolveAll() {
		if cred.Type == authentication.CredTypeNone || cred.Token == "" {
			continue
		}
		args = append(args, "-e", cred.EnvVar+"="+cred.Token)
	}
	for _, key := range toolPassthroughEnvVars {
		if v := os.Getenv(key); v != "" {
			args = append(args, "-e", key+"="+v)
		}
	}
	return args
}

// DockerEnvVars returns key=value env strings for container creation
// Via the Docker API (as opposed to the CLI flags from DockerEnvArgs).
func DockerEnvVars() []string {
	var envs []string
	for _, cred := range authentication.ResolveAll() {
		if cred.Type == authentication.CredTypeNone || cred.Token == "" {
			continue
		}
		envs = append(envs, cred.EnvVar+"="+cred.Token)
	}
	for _, key := range toolPassthroughEnvVars {
		if v := os.Getenv(key); v != "" {
			envs = append(envs, key+"="+v)
		}
	}
	return envs
}
