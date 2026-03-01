FROM ubuntu:24.04

# Avoid interactive prompts during package installation.
ENV DEBIAN_FRONTEND=noninteractive

# Install base utilities and Node.js 20 (required for Claude Code CLI).
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    jq \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && npm install -g @anthropic-ai/claude-code \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create standard mount points.
RUN mkdir -p /workspace /mind /universe

WORKDIR /workspace

# Container stays alive; the Architect execs Claude Code into it.
ENTRYPOINT ["sleep", "infinity"]
