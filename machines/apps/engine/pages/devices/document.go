package devices

import "github.com/jterrazz/studio-station/apps/engine/devices"

type Document struct {
	// value: the read's --json document: its schema, then the listing.
	Schema string `json:"schema"`
	devices.Listing
}
