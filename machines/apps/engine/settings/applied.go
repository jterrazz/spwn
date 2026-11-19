package settings

import "time"

type Applied struct {
	// value: what station last wrote for a setting, the digest of it and
	// when, so a change made since is seen.
	Setting string    `json:"setting"`
	Digest  string    `json:"digest"`
	At      time.Time `json:"at"`
}
