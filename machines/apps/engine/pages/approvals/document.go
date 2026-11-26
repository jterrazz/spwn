package approvals

import "github.com/jterrazz/studio-station/apps/engine/approvals"

type Document struct {
	// value: the read's --json document: its schema, then the listing.
	Schema string `json:"schema"`
	approvals.Listing
}
