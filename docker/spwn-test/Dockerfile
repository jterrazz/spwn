FROM ubuntu:26.04

RUN apt-get update && apt-get install -y \
    bash coreutils findutils grep sed gawk \
    ca-certificates curl wget git jq python3 python3-pip sudo \
    nodejs npm \
    && rm -rf /var/lib/apt/lists/*

RUN ln -sf /usr/bin/python3 /usr/local/bin/python

COPY claude/mock.sh /usr/local/bin/claude
COPY codex/mock.sh /usr/local/bin/codex
RUN chmod +x /usr/local/bin/claude /usr/local/bin/codex

RUN useradd -m -s /bin/bash spwn \
    && echo "spwn ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

RUN mkdir -p /agents /world /home/spwn/.claude \
    && echo '{"hasCompletedOnboarding":true,"projects":{"/workspaces":{"hasTrustDialogAccepted":true},"/home/spwn":{"hasTrustDialogAccepted":true}}}' > /home/spwn/.claude.json \
    && echo '{"skipDangerousModePermissionPrompt":true}' > /home/spwn/.claude/settings.json \
    && chown -R spwn:spwn /agents /world /home/spwn /home/spwn/.claude.json /home/spwn/.claude

USER spwn
WORKDIR /home/spwn

ENTRYPOINT ["sleep", "infinity"]
