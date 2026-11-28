package platform

// MindLayers lists the persistence layers every agent has as
// directories on disk: playbooks, journal. Identity is a single
// SOUL.md file at the agent root, a file and not a layer, because every
// agent has exactly one soul.
//
// Skills are not a layer of the Mind: they are dependencies (the
// `skill/` refs of agent.yaml and the SKILL.md files tools ship), which
// the build pipeline injects into /world/skills/ at image time.
//
// Knowledge is not a layer either: it is about the domain, not the
// agent, so it lives on the world, at its knowledge: path, bind-mounted
// into the container at /world/knowledge/.
var MindLayers = []string{"playbooks", "journal"}

// SoulFileName is the agent's identity file, the one place that
// answers "who is this agent". Lives at <agent>/SOUL.md.
const SoulFileName = "SOUL.md"
