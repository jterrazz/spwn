FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    bash coreutils findutils grep sed gawk \
    ca-certificates curl wget git jq \
    python3 make gcc g++ sudo docker.io \
    && rm -rf /var/lib/apt/lists/*

# Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# spwn CLI (God mode — manages the universe from inside)
RUN curl -fsSL https://spwn.sh/install.sh | bash || true

# Non-root user with Docker group access
RUN useradd -m -s /bin/bash spwn \
    && echo "spwn ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers \
    && usermod -aG docker spwn

# Create mount points + Claude Code config (onboarding + workspace trust)
RUN mkdir -p /workspace /mind /universe /world /home/spwn/.claude /home/spwn/.spwn \
    && echo '{"hasCompletedOnboarding":true,"projects":{"/workspace":{"hasTrustDialogAccepted":true},"/home/spwn":{"hasTrustDialogAccepted":true}}}' > /home/spwn/.claude.json \
    && echo '{"skipDangerousModePermissionPrompt":true}' > /home/spwn/.claude/settings.json \
    && chown -R spwn:spwn /workspace /mind /universe /world /home/spwn /home/spwn/.claude.json /home/spwn/.claude /home/spwn/.spwn

USER spwn
WORKDIR /home/spwn

VOLUME ["/workspace", "/mind", "/universe", "/world"]
ENTRYPOINT ["sleep", "infinity"]
