package darwin

import "testing"

func TestBatteryHealthIsTheCapacityItHoldsOverTheOneItWasBuiltFor(t *testing.T) {
	cases := []struct {
		name         string
		full, design int
		want         int
	}{
		{"a worn laptop's recorded BatteryData, which System Information shows as 84 %", 7319, 8694, 84},
		{"a new battery that holds more than its design reads 100", 4700, 4563, 100},
		{"no capacity read is no health", 0, 8694, 0},
		{"no design read is no health", 7319, 0, 0},
	}
	for _, c := range cases {
		if got := BatteryHealth(c.full, c.design); got != c.want {
			t.Errorf("%s: BatteryHealth(%d, %d) = %d; want %d", c.name, c.full, c.design, got, c.want)
		}
	}
}
