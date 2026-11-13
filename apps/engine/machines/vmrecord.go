package machines

import (
	"strconv"
	"strings"
	"time"
)

const (
	WantRunning = "running"
	WantStopped = "stopped"
	NetworkNAT  = "nat"
	Bridged     = "bridged"
)

func ParseResources(cpus int, memory, disk string) (Resources, error) {
	// invariant: --memory counts in binary units and --disk in decimal ones, as
	// the hypervisor takes them; a size says its unit, and none keeps the default.
	memoryBytes, err := parseSize("memory", memory, 1024)
	if err != nil {
		return Resources{}, err
	}
	diskBytes, err := parseSize("disk", disk, 1000)
	return Resources{CPUs: cpus, MemoryBytes: memoryBytes, DiskBytes: diskBytes}, err
}

func parseSize(flag, value string, base int64) (int64, error) {
	if value == "" {
		return 0, nil
	}
	text, scale := strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(value)), "B"), int64(0)
	if text != "" {
		if exponent := strings.IndexByte("KMGT", text[len(text)-1]); exponent >= 0 {
			scale = base
			for range exponent {
				scale *= base
			}
			text = text[:len(text)-1]
		}
	}
	if _, err := strconv.ParseFloat(text, 64); err == nil && scale == 0 {
		return 0, &SizeNeedsUnitError{Flag: flag, Value: value}
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil || n <= 0 {
		return 0, &BadSizeError{Flag: flag, Value: value}
	}
	return int64(n * float64(scale)), nil
}

type VMRecord struct {
	Name, Image, Version string
	Lifetime             Lifetime
	Want, Network, Check string
	Env                  map[string]string
	Resources            Resources
	CreatedAt            time.Time
	Birth                *Birth
	// value: the agents a workspace held when it last stopped.
	Agents []string
	// value: the Machinefile it was made from; empty for none.
	Template string
	// value: the folders of its Mac its last start shared.
	Mounts []Mount
}
