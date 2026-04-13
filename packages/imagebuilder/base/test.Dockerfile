FROM ubuntu:24.04

# Minimal test base — mock tools only
RUN apt-get update && apt-get install -y ca-certificates sudo \
    && rm -rf /var/lib/apt/lists/*

# Non-root user (matching production image)
RUN useradd -m -s /bin/bash spwn \
    && echo "spwn ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Create mount points
RUN mkdir -p /workspace /mind /universe /world /home/spwn/.spwn \
    && chown -R spwn:spwn /workspace /mind /universe /world /home/spwn /home/spwn/.spwn

# NOTE: USER, WORKDIR, VOLUME, ENTRYPOINT are added by the generator
