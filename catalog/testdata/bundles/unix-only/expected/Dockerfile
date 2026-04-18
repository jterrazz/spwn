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
    findutils \
    gawk \
    grep \
    jq \
    sed \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Final setup
RUN chown -R spwn:spwn /home/spwn
USER spwn
WORKDIR /home/spwn
ENTRYPOINT ["sleep", "infinity"]
