package agent

import (
	"fmt"

	"spwn.sh/apps/cli/cliproject"
)

func findWorldForAgent(agentName string) (string, error) {
	p, err := cliproject.Require()
	if err != nil {
		return "", err
	}
	for name, w := range p.Manifest.Worlds {
		for _, a := range w.Agents {
			if a == agentName {
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("agent %q is not in any world in spwn.yaml", agentName)
}
