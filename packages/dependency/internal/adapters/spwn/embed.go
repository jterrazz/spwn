package spwn

import "io/fs"

// EmbedFS returns a read-only view of the embedded catalog tree
// rooted at the logical catalog top — callers see <slug>/ entries
// at the root, hiding the on-disk content/ mirror. Exposed so
// external test packages + tooling can walk the bytes that ship in
// the compiled binary without duplicating the embed list.
func EmbedFS() fs.FS {
	sub, err := fs.Sub(catalogFS, contentRoot)
	if err != nil {
		// Embed always matches content/ at build time; this panic
		// would only fire if the mirror is empty (go generate
		// didn't run) — fail loudly.
		panic("spwn adapter: missing content/ mirror (run `go generate ./packages/dependency/...`)")
	}
	return sub
}
