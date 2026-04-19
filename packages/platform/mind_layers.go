package platform

// MindLayers lists the persistence layers every agent has as
// directories on disk: playbooks, journal. Identity used to be a
// separate layer (SOUL.md etc.) but was collapsed into a single
// SOUL.md file at the agent root in 2026-04 — a file, not a layer,
// because every agent has exactly one soul.
//
// Skills used to be a third Mind layer but were retired in 2026-04:
// the only "skills" concept left is dependency-driven (the `skill:`
// ref scheme in agent.yaml + SKILL.md files shipped by tools), which
// the build pipeline injects into /world/skills/ at image time. A
// per-agent runtime-writable skills/ directory was redundant with
// that and semantically overloaded the word.
//
// Knowledge moved out of the Mind in the same window: it's
// environmental (about the domain), not about the agent's personality,
// so it now lives on the world at spwn/worlds/<name>/knowledge/ and
// is bind-mounted into the container at /world/knowledge/.
var MindLayers = []string{"playbooks", "journal"}

// SoulFileName is the agent's identity file — the one place that
// answers "who is this agent". Lives at <agent>/SOUL.md.
const SoulFileName = "SOUL.md"
