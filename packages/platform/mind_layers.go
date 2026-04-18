package platform

// MindLayers lists the three persistence layers every agent has as
// directories on disk: skills, playbooks, journal. Identity used to
// be a fourth layer directory (SOUL.md etc.) but was
// collapsed into a single SOUL.md file at the agent root in 2026-04 —
// a file, not a layer, because every agent has exactly one soul and
// there's no benefit to a directory over a file.
//
// Knowledge moved out of the Mind earlier (also 2026-04): it's
// environmental (about the domain), not about the agent's personality,
// so it now lives on the world at spwn/worlds/<name>/knowledge/ and
// is bind-mounted into the container at /world/knowledge/.
var MindLayers = []string{"skills", "playbooks", "journal"}

// SoulFileName is the agent's identity file — the one place that
// answers "who is this agent". Lives at <agent>/SOUL.md.
const SoulFileName = "SOUL.md"
