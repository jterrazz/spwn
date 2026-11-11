package machines

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

const (
	WantRunning = "running"
	WantStopped = "stopped"
	NetworkNAT  = "nat"
	Bridged     = "bridged"
)

func ProfileEnv(profiles []catalog.Profile, reserved []Reserved) map[string]string {
	env := map[string]string{}
	publicName := ""
	for _, r := range reserved {
		publicName = r.PublicName
		for _, p := range profiles {
			if p.Name != r.Profile {
				continue
			}
			for _, e := range p.Env {
				env[e.Name] = render(e.Value, r)
			}
		}
	}
	if publicName != "" {
		env[catalog.PublicNameEnv.Name] = render(catalog.PublicNameEnv.Value, Reserved{PublicName: publicName})
	}
	return env
}

func render(template string, r Reserved) string {
	return strings.NewReplacer("{url}", r.URL, "{host}", r.PublicName, "{port}", strconv.Itoa(r.MacPort)).Replace(template)
}

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
}
