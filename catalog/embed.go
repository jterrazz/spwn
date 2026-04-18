package catalog

import (
	"io/fs"

	"spwn.sh/packages/dependency"
)

// EmbedFS returns a read-only view of the embedded catalog tree.
// Exposed so external test packages + tooling can walk the bytes
// that ship in the compiled binary without duplicating the embed
// list. Read-only by interface — writes are impossible through
// fs.FS.
func EmbedFS() fs.FS { return catalogFS }

// EntrySchema parses the spwn.yaml of the catalog entry at the
// given slug and returns the shared dependency.Schema. Returns
// the loader's os.PathError when the slug does not exist.
func EntrySchema(slug string) (*dependency.Schema, error) {
	return loadEntrySchema(slug)
}
