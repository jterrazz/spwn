package darwin

func BatteryHealth(full, design int) int {
	// outside: AppleSmartBattery's BatteryData holds the capacity the battery
	// holds now (NominalChargeCapacity) and the one it was built for
	// (DesignCapacity), in mAh; older macOS put AppleRawMaxCapacity and
	// DesignCapacity at its top. Their ratio, rounded and capped at 100, is
	// the Maximum Capacity System Information shows.
	if full <= 0 || design <= 0 {
		return 0
	}
	return min((full*100+design/2)/design, 100)
}
