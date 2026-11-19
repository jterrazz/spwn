package size

import "strconv"

func Bytes(n int64) string {
	// invariant: decimal units, as Finder, df -H and tart print sizes, so a size
	// station shows reads the same as the one the owner checks it against.
	const unit = 1000
	if n < unit {
		return strconv.FormatInt(max(n, 0), 10) + " B"
	}
	value, suffix := float64(n)/unit, 0
	for value >= unit-0.05 && suffix < len(suffixes)-1 {
		value /= unit
		suffix++
	}
	// value: three figures at most, as the vision prints sizes: 388 GB,
	// 73.3 GB, 5.8 GB.
	digits := 1
	if value >= 99.95 {
		digits = 0
	}
	return strconv.FormatFloat(value, 'f', digits, 64) + " " + string(suffixes[suffix]) + "B"
}

const suffixes = "KMGTPE"
