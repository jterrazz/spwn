package runtime

import (
	"fmt"
	"strings"
)

// GenerateDockerfile creates a Dockerfile for the given runtime adapter.
// This enables spwn to dynamically build container images for any supported runtime.
func GenerateDockerfile(rt Runtime) string {
	var b strings.Builder

	// Base image
	fmt.Fprintf(&b, "FROM %s\n\n", rt.BaseImage())

	// System packages (apt-get)
	pkgs := rt.SystemPackages()
	if len(pkgs) > 0 {
		b.WriteString("RUN apt-get update && apt-get install -y --no-install-recommends \\\n")
		for _, pkg := range pkgs {
			fmt.Fprintf(&b, "    %s \\\n", pkg)
		}
		b.WriteString("    && rm -rf /var/lib/apt/lists/*\n\n")
	}

	// Runtime installation
	for _, cmd := range rt.InstallCommands() {
		fmt.Fprintf(&b, "RUN %s\n", cmd)
	}
	b.WriteString("\n")

	// Non-root user (security — many runtimes refuse root)
	b.WriteString("# Non-root user\n")
	b.WriteString("RUN useradd -m -s /bin/bash spwn 2>/dev/null || true \\\n")
	b.WriteString("    && echo 'spwn ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers 2>/dev/null || true\n\n")

	// Mount points
	b.WriteString("# Mount points\n")
	b.WriteString("RUN mkdir -p /workspace /mind /universe /world \\\n")
	b.WriteString("    && chown -R spwn:spwn /workspace /mind /universe /world 2>/dev/null || true\n\n")

	// Switch to non-root
	b.WriteString("USER spwn\n")
	b.WriteString("WORKDIR /home/spwn\n\n")

	// Volumes
	b.WriteString("VOLUME [\"/workspace\", \"/mind\", \"/universe\", \"/world\"]\n")
	b.WriteString("ENTRYPOINT [\"sleep\", \"infinity\"]\n")

	return b.String()
}
