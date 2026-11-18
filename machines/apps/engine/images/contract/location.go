package contract

type ImageLocation struct {
	Hypervisor string
	// value: opaque to everyone but the hypervisor that fetches it: an OCI
	// reference to pull, or the name of an image disk it already holds.
	Ref string
}
