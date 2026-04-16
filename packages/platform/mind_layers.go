package platform

// MindLayers lists the five persistence layers every agent has:
// identity, skills, knowledge, playbooks, journal. Directory names
// under <agent>/ on disk. Kept in paths because it's a filesystem
// naming convention shared across multiple consumers (agent, mind,
// evolution).
var MindLayers = []string{"identity", "skills", "knowledge", "playbooks", "journal"}
