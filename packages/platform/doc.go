// Package platform is the foundation layer: directory constants,
// ID generation, bundled name lists, host-platform conventions.
//
// Every other spwn package asks platform for "where does this
// thing live on disk?" and "what should I call a new world or
// agent?" The functions respect SPWN_HOME (defaults to ~/.spwn)
// and a project-root override (SetProjectRoot, consulted by
// AgentsDir / SkillsDir / LocalStateDir so project-aware paths
// resolve under the project instead of the user dir).
//
// platform has zero spwn dependencies — stdlib only — so it's
// import-cycle-proof at the base of the layer graph.
package platform
