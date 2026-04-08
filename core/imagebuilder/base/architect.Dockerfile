FROM debian:bookworm-slim

# Minimal base — tools are added by imagebuilder
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl gnupg sudo \
    && rm -rf /var/lib/apt/lists/*

# Non-root user for Claude Code (refuses --dangerously-skip-permissions as root)
RUN useradd -m -s /bin/bash architect

# Universe data dir — mounted from host
RUN mkdir -p /me /universe && chmod 777 /universe && chown -R architect:architect /me
ENV SPWN_HOME=/universe
WORKDIR /me

ENTRYPOINT ["sleep", "infinity"]
