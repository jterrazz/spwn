package catalog

import (
	"testing"
	"testing/fstest"
)

func TestABuiltInImageIsSizedByItsMachinefileAlone(t *testing.T) {
	files := fstest.MapFS{
		"machines/base-ubuntu.machine.yaml": {Data: []byte(`machinefile: station.machinefile/v1
name: base-ubuntu
machine: { kind: workspace, system: { os: ubuntu, version: '24.04' } }
vm: { resources: { cpus: 6, memory: 12G, disk: 1T } }
`)},
		"machines/base-macos.machine.yaml": {Data: []byte(`machinefile: station.machinefile/v1
name: base-macos
machine: { kind: workspace, system: { os: macos } }
vm: { resources: { cpus: 3, memory: 6G, disk: 90G } }
`)},
	}
	images, err := sizedImages(files)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]ImageDefaults{
		LinuxImage: {CPUs: 6, MemoryBytes: 12 << 30, DiskBytes: 1e12},
		MacOSImage: {CPUs: 3, MemoryBytes: 6 << 30, DiskBytes: 90e9},
	}
	for _, img := range images {
		if img.Defaults != want[img.Name] {
			t.Errorf("%s sized %+v; want its file's %+v", img.Name, img.Defaults, want[img.Name])
		}
	}
	if len(images) != len(want) {
		t.Errorf("%d built-in images; want %d", len(images), len(want))
	}
	if _, err := sizedImages(fstest.MapFS{}); err == nil {
		t.Error("no base-ubuntu file: want the catalog refused, never a size of nowhere")
	}
}
