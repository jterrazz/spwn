FROM ubuntu:24.04

# Minimal base — tools are added by imagebuilder
RUN apt-get update && apt-get install -y ca-certificates sudo \
    && rm -rf /var/lib/apt/lists/*

# Non-root user
RUN useradd -m -s /bin/bash spwn \
    && echo "spwn ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Create mount points
RUN mkdir -p /workspace /mind /universe /world /home/spwn/.spwn \
    && chown -R spwn:spwn /workspace /mind /universe /world /home/spwn /home/spwn/.spwn

# NOTE: USER, WORKDIR, ENTRYPOINT are added by the generator
# after all tool installs complete (tools need root access).

LABEL sh.spwn.image-version="v-test"

# Packages (merged from all tools)
RUN apt-get update && apt-get install -y \
    bash \
    coreutils \
    curl \
    docker.io \
    findutils \
    gawk \
    grep \
    jq \
    python3 \
    python3-pip \
    sed \
    wget \
    && rm -rf /var/lib/apt/lists/*

# spwn:cli (platform)
RUN curl -fsSL https://spwn.sh/install.sh | bash || true

# spwn:node (sdk)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
RUN apt-get install -y nodejs && rm -rf /var/lib/apt/lists/*

# spwn:claude-code (runtime)
RUN curl -fsSL https://claude.ai/install.sh | bash
RUN cp -L /root/.local/bin/claude /usr/local/bin/claude
RUN chmod +x /usr/local/bin/claude
RUN rm -rf /root/.local /root/.claude

# spwn:mempalace (tool)
RUN pip install --no-cache-dir --break-system-packages mempalace

# spwn:qmd (tool)
RUN npm install -g @tobilu/qmd

# spwn:architect (platform)
COPY tools/spwn:architect/usr/local/bin/architect-entrypoint.sh /usr/local/bin/architect-entrypoint.sh
RUN chmod +x /usr/local/bin/architect-entrypoint.sh

# Skills (copied from build context)
COPY skills/ /

# Final setup
RUN chown -R spwn:spwn /home/spwn
USER spwn
WORKDIR /home/spwn
ENTRYPOINT ["sleep", "infinity"]
