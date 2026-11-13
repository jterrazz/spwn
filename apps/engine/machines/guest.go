package machines

type GuestOnMacError struct{}

func (*GuestOnMacError) Error() string { return "station guest runs only inside a VM" }
