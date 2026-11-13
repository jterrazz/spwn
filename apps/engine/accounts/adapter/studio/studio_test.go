package studio_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/studio"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the OS CLI's store as `studio credentials export` answers: one JSON
// object of the names asked, a name it does not keep failing the export.
func store(kept map[string]string) local.Runner {
	return local.Played(func(c local.Cmd) (local.Output, error) {
		if len(c.Args) < 2 || c.Args[0] != "credentials" || c.Args[1] != "export" {
			return local.Output{ExitCode: 97, Stderr: "unexpected: " + strings.Join(c.Args, " ")}, nil
		}
		out := map[string]string{}
		for _, n := range c.Args[2:] {
			v, ok := kept[n]
			if !ok {
				return local.Output{ExitCode: 1, Stderr: "credential " + n + " not found"}, nil
			}
			out[n] = v
		}
		data, _ := json.Marshal(out)
		return local.Output{Stdout: string(data) + "\n"}, nil
	}, "studio")
}

func TestSecretSource(t *testing.T) {
	contracttest.SecretSource(t, func(*testing.T) contracttest.SecretSourceSubject {
		kept := map[string]string{"session:a": "value-a", "operator:claude:oauth-token": "value-token"}
		return contracttest.SecretSourceSubject{Source: studio.New(store(kept), studio.Options{}), Kept: kept}
	})
}
