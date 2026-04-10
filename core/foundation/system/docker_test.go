package system

import (
	"context"
	"testing"
)

func TestCheckDocker_PlatformAlwaysSet(t *testing.T) {
	st := CheckDocker(context.Background())
	if st.Platform == "" {
		t.Fatal("Platform should always be set")
	}
}

func TestDockerStatus_Summary(t *testing.T) {
	cases := []struct {
		name string
		st   DockerStatus
		want string
	}{
		{"not installed", DockerStatus{}, "not installed"},
		{"installed not running", DockerStatus{Installed: true}, "not running"},
		{"running no version", DockerStatus{Installed: true, Running: true}, "running"},
		{"running with version", DockerStatus{Installed: true, Running: true, Version: "27.0.1"}, "running (v27.0.1)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.st.Summary(); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestDockerStatus_OK(t *testing.T) {
	if (DockerStatus{}).OK() {
		t.Error("empty status should not be OK")
	}
	if (DockerStatus{Installed: true}).OK() {
		t.Error("installed-only should not be OK")
	}
	if !(DockerStatus{Installed: true, Running: true}).OK() {
		t.Error("installed+running should be OK")
	}
}
