FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    git \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# Container-side Gate binary (Rust, static)
# Built separately via: cd container/gate && cargo build --release --target x86_64-unknown-linux-musl
# COPY universe-gate /usr/local/bin/gate

# Mount points — /gate is for the host-side Gate socket and bridge wrappers
VOLUME ["/workspace", "/mind", "/universe"]

# Default entrypoint. When the container-side Gate binary is included,
# change to: ENTRYPOINT ["gate", "--listen", "/gate/gate.sock"]
ENTRYPOINT ["sleep", "infinity"]
