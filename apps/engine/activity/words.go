package activity

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// invariant: SPEC §6.2's rules of the numbers: disk and network sizes are
// decimal, memory sizes binary, every CPU share of the whole machine.

func decimal(v float64) string {
	// value: one decimal, dropped when it is zero or the number is 100 or more.
	if v >= 100 || v == math.Trunc(v) {
		return strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	}
	s := strconv.FormatFloat(v, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

func Bytes[N int64 | uint64](n N) string {
	// value: a decimal size, as Finder shows one: 388 GB, 4.6 GB, 2.3 TB.
	f := float64(n)
	for _, u := range []struct {
		unit string
		of   float64
	}{{"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"KB", 1e3}} {
		if f >= u.of {
			return decimal(math.Round(f/u.of*10)/10) + " " + u.unit
		}
	}
	return strconv.FormatFloat(f, 'f', 0, 64) + " B"
}

func Memory(n int64) string {
	// value: a binary size, as macOS shows RAM: 16 GB is 16 GiB; under a
	// tenth of a GiB in MB.
	const gib, mib = 1 << 30, 1 << 20
	switch {
	case n <= 0:
		return "0 B"
	case n >= gib/10:
		return decimal(math.Round(float64(n)/gib*10)/10) + " GB"
	}
	return strconv.FormatInt(int64(math.Round(float64(n)/mib)), 10) + " MB"
}

func Rate(perSecond float64) string {
	// value: bytes a second: KB/s below 1 MB/s, MB/s above, one decimal under 10.
	if perSecond < 1e6 {
		return strconv.FormatFloat(math.Round(perSecond/1e3), 'f', 0, 64) + " KB/s"
	}
	return decimal(math.Round(perSecond/1e6*10)/10) + " MB/s"
}

func Percent(p float64) string {
	// value: a share of the whole machine; one decimal under 1 %.
	if p > 0 && p < 1 {
		return strconv.FormatFloat(p, 'f', 1, 64) + " %"
	}
	return strconv.FormatFloat(math.Round(p), 'f', 0, 64) + " %"
}

func Window(d time.Duration) string {
	// value: what a rate is the mean of, in words.
	switch {
	case d <= 0:
		return "no rate yet"
	case d < time.Minute:
		return strconv.Itoa(int(math.Round(d.Seconds()))) + " s"
	}
	return strconv.Itoa(int(math.Round(d.Minutes()))) + " min"
}

func Thermal(level int) string {
	// value: thermalpressurelevel 0 to 4 in the words of SPEC §6.2.
	switch {
	case level <= 0:
		return "Nominal"
	case level == 1:
		return "Fair"
	case level == ThermalSerious:
		return "Serious"
	}
	return "Critical"
}

func BatteryState(b Battery) string {
	switch {
	case b.OnBattery:
		return "on battery"
	case b.Charging:
		return "charging"
	}
	return "charged"
}

func Load(l [3]float64) string {
	// value: the load averages of 1, 5 and 15 min: "2.8 3.1 2.9".
	parts := make([]string, 0, len(l))
	for _, v := range l {
		parts = append(parts, strconv.FormatFloat(v, 'f', 1, 64))
	}
	return strings.Join(parts, " ")
}
