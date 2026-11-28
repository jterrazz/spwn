// Package dependency owns the tool.yaml manifest schema, ref
// parsing, filesystem loading, and the committed spwn.lock file.
//
// One shared schema backs both catalog entries (built-in,
// compiled into the spwn binary) and project-local tools under
// tools/<name>/. Parse() takes a Resolver (an FSResolver, over
// the embedded catalog or a project read through its root) and
// ParseOptions, and returns
// a *Parsed that downstream layers (image, project, validate)
// consume.
//
// The Ref kinds (Local, SpwnBuiltin, Registry) classify what a
// user-facing ref like "spwn:unix" or "github.com/foo/bar" means.
// ResolveTool and ResolveSkill answer whether a ref actually
// resolves to something on disk. Lockfile owns the line-oriented
// spwn.lock text format (plus a legacy-YAML fallback for migration).
//
// The package imports no spwn package but packages/platform, so every
// upstream layer can import it. The built-in tool catalog
// lives in the subpackage builtin, which embeds the generated copy of
// /catalog and is linked by no engine package.
package dependency
