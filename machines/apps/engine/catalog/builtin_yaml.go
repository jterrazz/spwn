//go:build catalog_yaml

package catalog

// value: the built-in catalog read from its files, for `make generate`,
// which compiles the table from them and so builds without it.
func builtinCatalog() (Catalog, error) { return Load(builtin, "packages") }
