FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    bash coreutils findutils grep sed gawk \
    ca-certificates curl wget git jq python3 python3-pip sudo \
    nodejs npm \
    && rm -rf /var/lib/apt/lists/*

# The default `spwn init` scaffold declares the spwn:python tool, whose probe
# is `command -v python`. ubuntu ships only python3, so alias python→python3 so
# the mock base can verify that tool (the real world base already provides it).
RUN ln -sf /usr/bin/python3 /usr/local/bin/python

# Runtime simulators: protocol-faithful stubs for the claude-code and
# codex CLIs so tool probing + one-shot exec succeed without shipping
# the real vendor binaries in the test image.
COPY claude/mock.sh /usr/local/bin/claude
COPY codex/mock.sh /usr/local/bin/codex
RUN chmod +x /usr/local/bin/claude /usr/local/bin/codex

# Non-root user (matching production image)
RUN useradd -m -s /bin/bash spwn \
    && echo "spwn ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Create mount points + Claude Code config (onboarding + workspace trust + permission skip)
RUN mkdir -p /agents /world /home/spwn/.claude \
    && echo '{"hasCompletedOnboarding":true,"projects":{"/workspaces":{"hasTrustDialogAccepted":true},"/home/spwn":{"hasTrustDialogAccepted":true}}}' > /home/spwn/.claude.json \
    && echo '{"skipDangerousModePermissionPrompt":true}' > /home/spwn/.claude/settings.json \
    && chown -R spwn:spwn /agents /world /home/spwn /home/spwn/.claude.json /home/spwn/.claude

USER spwn
WORKDIR /home/spwn

# NOTE: no VOLUME declarations here on purpose. spwn bind-mounts
# /world and /workspaces/<name> at runtime; /agents content is
# docker cp'd in. Declared VOLUMEs would auto-create an anonymous
# volume per container spawn that `docker rm` never cleans up.
ENTRYPOINT ["sleep", "infinity"]
