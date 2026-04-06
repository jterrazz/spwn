FROM ubuntu:24.04
LABEL sh.spwn.image-version="1.0.0"

RUN apt-get update && apt-get install -y \
    bash coreutils findutils grep sed gawk \
    ca-certificates curl wget git jq \
    python3 make gcc g++ sudo \
    && rm -rf /var/lib/apt/lists/*

# Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# Non-root user (Claude Code refuses --dangerously-skip-permissions as root)
RUN useradd -m -s /bin/bash spwn \
    && echo "spwn ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Create mount points + Claude Code config (onboarding + workspace trust)
RUN mkdir -p /workspace /mind /universe /world /home/spwn/.claude \
    && echo '{"hasCompletedOnboarding":true,"projects":{"/workspace":{"hasTrustDialogAccepted":true},"/home/spwn":{"hasTrustDialogAccepted":true}}}' > /home/spwn/.claude.json \
    && echo '{"skipDangerousModePermissionPrompt":true}' > /home/spwn/.claude/settings.json \
    && chown -R spwn:spwn /workspace /mind /universe /world /home/spwn /home/spwn/.claude.json /home/spwn/.claude

USER spwn
WORKDIR /home/spwn

VOLUME ["/workspace", "/mind", "/universe", "/world"]
ENTRYPOINT ["sleep", "infinity"]
