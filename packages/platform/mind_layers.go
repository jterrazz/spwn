package platform

// MindLayers lists the four persistence layers every agent has:
// identity, skills, playbooks, journal. Directory names under
// <agent>/ on disk. Kept in paths because it's a filesystem naming
// convention shared across multiple consumers (agent, mind,
// evolution).
//
// Knowledge used to be a fifth layer but moved to the world in
// 2026-04: it's environmental (about the domain), not about the
// agent's personality. See packages/architect/spawn.go for how
// project-committed knowledge under spwn/worlds/<name>/knowledge/
// is bind-mounted into the container at /world/knowledge/.
var MindLayers = []string{"identity", "skills", "playbooks", "journal"}
