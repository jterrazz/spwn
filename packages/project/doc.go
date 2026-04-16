// Package project owns the spwn project manifest and every static
// rule that guards it: parsing spwn.yaml, walking up from any
// subdirectory to locate the nearest manifest (Find), scaffolding
// a fresh project (Init), and running the validation rule engine
// (Validate) that backs `spwn check`.
//
// Also lives here: project-scoped entities that span multiple
// agents — Teams and Organizations (role hierarchies). These were
// part of packages/agent until the boundary audit surfaced that
// they're project-level concerns stored under ~/.spwn/teams and
// ~/.spwn/organizations.
//
// The internal/ subpackages (manifest, discovery, scaffold,
// validate, resolve) hold implementation details; external
// callers reach in through the public facade at the package
// root.
package project
