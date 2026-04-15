package ui

import "testing"

func TestSummariseInstruction(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "apt-get install",
			in:   "RUN apt-get update && apt-get install -y bash git python3",
			want: "Installing system packages",
		},
		{
			name: "npm install global claude",
			in:   "RUN npm install -g @anthropic-ai/claude-code",
			want: "Installing @anthropic-ai/claude-code",
		},
		{
			name: "npm install global qmd",
			in:   "RUN npm install -g @tobilu/qmd",
			want: "Installing @tobilu/qmd",
		},
		{
			name: "FROM base image",
			in:   "FROM ubuntu:24.04",
			want: "Pulling base image",
		},
		{
			name: "COPY skills",
			in:   "COPY skills /",
			want: "Copying files",
		},
		{
			name: "chown to user",
			in:   "RUN chown -R spwn:spwn /home/spwn",
			want: "Setting up user",
		},
		{
			name: "mkdir",
			in:   "RUN mkdir -p /workspace /world",
			want: "Creating directories",
		},
		{
			name: "LABEL",
			in:   `LABEL sh.spwn.image-version="abc123"`,
			want: "Labelling image",
		},
		{
			name: "ENV var",
			in:   "ENV PATH=/usr/local/bin",
			want: "Setting environment",
		},
		{
			name: "WORKDIR",
			in:   "WORKDIR /home/spwn",
			want: "Setting workdir",
		},
		{
			name: "VOLUME finalising",
			in:   `VOLUME ["/work", "/agents"]`,
			want: "Finalising",
		},
		{
			name: "unknown instruction",
			in:   "HEALTHCHECK CMD curl -f http://localhost/ || exit 1",
			want: "Running step",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := summariseInstruction(tc.in)
			if got != tc.want {
				t.Errorf("summariseInstruction(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestBuildProgressWriter_ParsesStepLines covers the end-to-end
// path: a Docker build stream (one fragment at a time, because the
// HTTP stream rarely matches line boundaries) arrives at the
// writer, and the stepper's label is updated exactly once per
// `Step N/M :` line with the summarised action.
func TestBuildProgressWriter_ParsesStepLines(t *testing.T) {
	s := &Stepper{isTTY: true}
	w := s.BuildProgressWriter("Building image").(*buildProgressWriter)

	// Feed the stream in a few chunks that split across line
	// boundaries. This exercises the bytes.Buffer half of the
	// writer - a naive line-based parser would miss partial lines.
	fragments := []string{
		"Step 1/3 : FROM ubuntu:24.04\n",
		"Step 2/3 : RUN apt-ge",
		"t install -y bash git\nStep 3/3 : CO",
		"PY skills /\n",
	}
	for _, f := range fragments {
		n, err := w.Write([]byte(f))
		if err != nil {
			t.Fatalf("Write(%q): %v", f, err)
		}
		if n != len(f) {
			t.Errorf("Write(%q) wrote %d, want %d", f, n, len(f))
		}
	}

	// The last `Step N/M :` line wins because each overwrites
	// the previous label in place.
	s.mu.Lock()
	got := s.label
	s.mu.Unlock()

	// Labels contain the base + the ANSI-wrapped [N/M] + action.
	// We strip colours before comparing.
	const want = "Building image [3/3] Copying files"
	if !containsVisible(got, want) {
		t.Errorf("spinner label = %q, want visible %q", got, want)
	}
}

// containsVisible returns true if haystack contains needle after
// stripping ANSI escape sequences. Copied inline to avoid pulling
// in a regex for such a small test.
func containsVisible(haystack, needle string) bool {
	// Dumb but correct: scan linearly, skipping over any `\x1b[...m`
	// escape runs when matching.
	visible := make([]byte, 0, len(haystack))
	for i := 0; i < len(haystack); i++ {
		if haystack[i] == 0x1b && i+1 < len(haystack) && haystack[i+1] == '[' {
			for i < len(haystack) && haystack[i] != 'm' {
				i++
			}
			continue
		}
		visible = append(visible, haystack[i])
	}
	return string(visible) != "" && indexOf(string(visible), needle) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
