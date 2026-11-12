package machines

import "strconv"

// value: a folder of the Mac shared live with one of its VMs, at a folder
// of the guest, each a full path once the create resolved its `~/`;
// read-only when the VM may not write to it.

type Mount struct {
	Computer string `json:"computer"`
	Guest    string `json:"guest"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

func MountTag(i int) string {
	// value: the tag the hypervisor shares the mount at index i under, and
	// the guest mounts it by.
	return "station-" + strconv.Itoa(i+1)
}
