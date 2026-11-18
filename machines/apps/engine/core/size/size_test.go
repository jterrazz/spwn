package size

import "testing"

func TestBytes(t *testing.T) {
	for n, want := range map[int64]string{
		-5:              "0 B",
		0:               "0 B",
		999:             "999 B",
		1000:            "1.0 KB",
		5_800_000_000:   "5.8 GB",
		73_300_000_000:  "73.3 GB",
		388_000_000_000: "388 GB",
		999_960_000:     "1.0 GB",
	} {
		if got := Bytes(n); got != want {
			t.Errorf("Bytes(%d) = %q, want %q", n, got, want)
		}
	}
}
