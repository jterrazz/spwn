// Package dependency owns the spwn.yaml manifest schema, ref
// parsing, filesystem loading, and the committed spwn.lock file.
//
// One shared schema backs both catalog entries (built-in,
// compiled into the spwn binary) and project-local tools under
// spwn/tools/<name>/. Parse() takes a Resolver (DirResolver for
// disk, EmbedResolver for go:embed) and ParseOptions, and returns
// a *Parsed that downstream layers (image, project, validate)
// consume.
//
// The Ref kinds (Local, SpwnBuiltin, Registry) classify what a
// user-facing ref like "@spwn/unix" or "github.com/foo/bar" means.
// ResolveTool and ResolveSkill answer whether a ref actually
// resolves to something on disk. Lockfile owns the line-oriented
// spwn.lock text format (plus a legacy-YAML fallback for migration).
//
// The package has zero spwn dependencies — only stdlib — so every
// upstream layer can import it.
package dependency
