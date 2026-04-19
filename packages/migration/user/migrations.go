// Package user contains the user-scoped migration catalogue
// (numbered .go files) plus a type alias over packages/migration.
// Migrations advance the ~/.spwn on-disk schema as the CLI evolves.
package user

import "spwn.sh/packages/migration"

// Migration is an alias for convenience.
type Migration = migration.Migration
