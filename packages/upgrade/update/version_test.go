package update

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in      string
		major   int
		minor   int
		patch   int
		pre     string
		isDev   bool
		wantErr bool
	}{
		{in: "v1.2.3", major: 1, minor: 2, patch: 3},
		{in: "1.2.3", major: 1, minor: 2, patch: 3},
		{in: "v0.11.0", minor: 11},
		{in: "v1.2.3-beta.1", major: 1, minor: 2, patch: 3, pre: "beta.1"},
		{in: "v1.0.0-nightly.20260405", major: 1, pre: "nightly.20260405"},
		{in: "dev", isDev: true},
		{in: "", isDev: true},
		{in: "v1.2", wantErr: true},
		{in: "not-a-version", wantErr: true},
		{in: "v1.x.3", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			v, err := ParseVersion(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", v)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.IsDev() != tt.isDev {
				t.Errorf("IsDev() = %v, want %v", v.IsDev(), tt.isDev)
			}
			if tt.isDev {
				return
			}
			if v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch {
				t.Errorf("parsed %d.%d.%d, want %d.%d.%d", v.Major, v.Minor, v.Patch, tt.major, tt.minor, tt.patch)
			}
			if v.Prerelease != tt.pre {
				t.Errorf("prerelease = %q, want %q", v.Prerelease, tt.pre)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	mk := func(s string) Version {
		v, err := ParseVersion(s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	tests := []struct {
		a, b string
		want int
	}{
		// equal
		{"v1.2.3", "v1.2.3", 0},
		{"1.2.3", "v1.2.3", 0},
		// major/minor/patch ordering
		{"v1.2.3", "v1.2.4", -1},
		{"v1.2.4", "v1.2.3", 1},
		{"v1.3.0", "v1.2.9", 1},
		{"v2.0.0", "v1.99.99", 1},
		// dev is always older than anything real, equal to itself
		{"dev", "v1.0.0", -1},
		{"v1.0.0", "dev", 1},
		{"dev", "dev", 0},
		// prerelease rules: pre is older than its release
		{"v1.0.0-beta.1", "v1.0.0", -1},
		{"v1.0.0", "v1.0.0-beta.1", 1},
		// prerelease string comparison
		{"v1.0.0-beta.1", "v1.0.0-beta.2", -1},
		{"v1.0.0-alpha.1", "v1.0.0-beta.1", -1},
	}
	for _, tt := range tests {
		got := mk(tt.a).Compare(mk(tt.b))
		if got != tt.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsPrerelease(t *testing.T) {
	tests := []struct {
		v    string
		want bool
	}{
		{"v1.2.3", false},
		{"v1.2.3-beta.1", true},
		{"v1.2.3-rc.0", true},
		{"dev", false},
	}
	for _, tt := range tests {
		v, _ := ParseVersion(tt.v)
		if v.IsPrerelease() != tt.want {
			t.Errorf("IsPrerelease(%q) = %v, want %v", tt.v, v.IsPrerelease(), tt.want)
		}
	}
}

func TestString_Roundtrip(t *testing.T) {
	tests := []string{"v1.2.3", "v1.2.3-beta.1", "v0.0.0", "dev"}
	for _, in := range tests {
		v, err := ParseVersion(in)
		if err != nil {
			t.Fatal(err)
		}
		got := v.String()
		want := in
		if in == "" {
			want = "dev"
		}
		if got != want {
			t.Errorf("String(%q) = %q, want %q", in, got, want)
		}
	}
}
