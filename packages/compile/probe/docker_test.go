package probe

import (
	"context"
	"errors"
	"testing"
)

func TestDockerStatus_OK(t *testing.T) {
	if (DockerStatus{}).OK() {
		t.Error("empty status should not be OK")
	}
	if !(DockerStatus{Running: true}).OK() {
		t.Error("running status should be OK")
	}
}

func TestDockerStatus_Summary(t *testing.T) {
	cases := []struct {
		name string
		st   DockerStatus
		want string
	}{
		{"not running", DockerStatus{}, "not running"},
		{"running bare", DockerStatus{Running: true}, "running"},
		{"running with version", DockerStatus{Running: true, Version: "27.0.1"}, "running (v27.0.1)"},
		{"running with runtime+version", DockerStatus{Running: true, Runtime: "OrbStack", Version: "27.0.1"}, "running (OrbStack v27.0.1)"},
		{"running with runtime only", DockerStatus{Running: true, Runtime: "Colima"}, "running (Colima)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.st.Summary(); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestInferRuntime(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"unix:///Users/x/.orbstack/run/docker.sock", "OrbStack"},
		{"unix:///Users/x/.colima/default/docker.sock", "Colima"},
		{"unix:///Users/x/.rd/docker.sock", "Rancher Desktop"},
		{"unix:///Users/x/.lima/default/sock/docker.sock", "Lima"},
		{"unix:///run/user/1000/podman/podman.sock", "Podman"},
		{"unix:///Users/x/.docker/run/docker.sock", "Docker Desktop"},
		{"unix:///var/run/docker.sock", "Docker"},
		{"tcp://example.com:2376", "Docker"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			if got := inferRuntime(tc.host); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestUnixPath(t *testing.T) {
	if path, ok := unixPath("unix:///var/run/docker.sock"); !ok || path != "/var/run/docker.sock" {
		t.Errorf("unix path: got (%q,%v)", path, ok)
	}
	if _, ok := unixPath("tcp://example.com:2376"); ok {
		t.Error("tcp host should not match")
	}
	if _, ok := unixPath(""); ok {
		t.Error("empty host should not match")
	}
}

func TestHumanizeDaemonError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "missing socket file",
			err:  errors.New("Cannot connect: dial unix /var/run/docker.sock: connect: no such file or directory"),
			want: "no docker daemon listening on the configured socket",
		},
		{
			name: "connection refused",
			err:  errors.New("dial unix: connect: connection refused"),
			want: "no docker daemon listening on the configured socket",
		},
		{
			name: "permission denied",
			err:  errors.New("Got permission denied while trying to connect to the Docker daemon socket"),
			want: "permission denied talking to the docker socket",
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			want: "docker daemon did not respond in time",
		},
		{
			name: "nil error",
			err:  nil,
			want: "docker daemon is not reachable",
		},
		{
			name: "unknown",
			err:  errors.New("something weird happened"),
			want: "something weird happened",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := humanizeDaemonError(tc.err); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestCandidateHosts_Nonempty(t *testing.T) {
	hosts := candidateHosts()
	if len(hosts) == 0 {
		t.Fatal("expected at least one candidate host")
	}
	// First entries (when home is available) should be per-user.
	for _, h := range hosts {
		if h == "" {
			t.Errorf("empty host in list")
		}
	}
}

func TestCheckDocker_AlwaysReturnsPlatform(t *testing.T) {
	st := CheckDocker(context.Background())
	if st.Platform == "" {
		t.Error("Platform should always be populated")
	}
}
