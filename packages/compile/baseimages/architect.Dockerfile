FROM debian:bookworm-slim

# Minimal base — tools are added by the compile package
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl gnupg sudo \
    && rm -rf /var/lib/apt/lists/*

# Non-root user for Claude Code (refuses --dangerously-skip-permissions as root)
RUN useradd -m -s /bin/bash architect

# Universe data dir — mounted from host
RUN mkdir -p /me /universe && chmod 777 /universe && chown -R architect:architect /me
ENV SPWN_HOME=/universe

# NOTE: the compile package inserts tool installs here (Node.js, Claude Code, Docker CLI, etc.)
# Then the architect-specific COPY and entrypoint directives are appended by BuildArchitectImage.
