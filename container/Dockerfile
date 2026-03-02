FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    git \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Claude Code CLI
RUN curl -fsSL https://spwn.sh/universe.sh | bash

# Mount points
VOLUME ["/workspace", "/mind", "/universe"]

ENTRYPOINT ["sleep", "infinity"]
