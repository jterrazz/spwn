package catalog

import (
	"testing"
	"testing/fstest"
)

func TestABuiltInImageIsSizedByItsMachinefileAlone(t *testing.T) {
	files := fstest.MapFS{"machines/station-linux.machine.yaml": {Data: []byte(`machinefile: station.machinefile/v1
name: station-linux
machine: { kind: workspace, system: { os: ubuntu, version: '24.04' } }
vm: { resources: { cpus: 6, memory: 12G, disk: 1T } }
`)}}
	images, err := sizedImages(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, img := range images {
		switch img.Name {
		case LinuxImage:
			if img.Defaults != (ImageDefaults{CPUs: 6, MemoryBytes: 12 << 30, DiskBytes: 1e12}) {
				t.Errorf("station-linux sized %+v; want its file's 6 CPUs, 12 GiB and 1 TB", img.Defaults)
			}
		case "macos":
			if img.Defaults.MemoryBytes != 8<<30 {
				t.Errorf("macos sized %+v; want its own 8 GiB, it having no file", img.Defaults)
			}
		}
	}
	if _, err := sizedImages(fstest.MapFS{}); err == nil {
		t.Error("no station-linux file: want the catalog refused, never a size of nowhere")
	}
}
