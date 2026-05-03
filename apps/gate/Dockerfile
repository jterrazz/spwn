# spwn-gate — host-side credential broker / element bridge.
#
# Three layers inside the container:
#
#   1. spwn-gate (Go binary) — entrypoint. Hosts the public HTTP API
#      on :9000 (cookie sync + MCP routing). Spawns and supervises
#      the gate-browser sidecar; will spawn catalog-tool subprocesses
#      once Phase 2 of the generic refactor lands.
#
#   2. gate-browser (Node sidecar, internal :9001) — Playwright
#      Chromium pool. Loads session cookies per provider from
#      /credentials/<p>/cookies.json. Catalog tools call its HTTP
#      API to drive scraped sites.
#
#   3. Element handlers (Go in-tree for now: notion proxy, gws
#      gmail/gcal, x via x-browse). Migrating to catalog tools.
#
# The Go binary is cross-compiled on the host (so go.work + every
# workspace module resolve correctly) and COPY'd in. Same pattern as
# packages/architect builds its image.
#
# Build context expected: a directory containing the linux binary at
# ./spwn-gate plus ./browser/ (the Node sidecar dir). packages/gate/
# lifecycle.go produces that.

FROM mcr.microsoft.com/playwright:v1.59.1-jammy

# jq for shell-side cookie inspection; gws for gmail/gcal elements;
# docker CLI so the spwn:dispatch tool can `spwn agent talk` (which
# shells docker exec) when /var/run/docker.sock is bind-mounted in.
# The Microsoft Playwright image ships browsers (in /ms-playwright)
# but NOT the `playwright` npm module — install it globally so the
# sidecar + tools can `require('playwright')`. SKIP_BROWSER_DOWNLOAD
# avoids re-downloading the chromium the base image already provides.
RUN apt-get update && apt-get install -y --no-install-recommends jq docker.io \
 && rm -rf /var/lib/apt/lists/* \
 && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm install -g playwright @googleworkspace/cli

ENV NODE_PATH=/usr/lib/node_modules

# Non-root user for the gate process.
RUN groupadd --system spwn && useradd --system --gid spwn --create-home --shell /bin/bash spwn

COPY spwn-gate /usr/local/bin/spwn-gate
COPY browser /opt/gate-browser
# @spwn/gate-tool — the Node SDK every catalog tool requires.
# Drop it under the global node_modules path so tools can resolve it
# via NODE_PATH=/usr/lib/node_modules without per-tool npm install.
COPY sdk /usr/lib/node_modules/@spwn/gate-tool
RUN chmod +x /usr/local/bin/spwn-gate /opt/gate-browser/index.js

USER spwn
WORKDIR /home/spwn

EXPOSE 9000

# Bind mounts (runtime, set by spwn gate start):
#   /credentials  ← ~/.spwn/credentials  (rw)  — gate is the cred broker
#   /gate         ← ~/.spwn/gate         (rw)  — persistent state
#
# Cookie sources (browser extension writes; sidecar reads):
#   /credentials/<provider>/cookies.json  — pushed by spwn-cookie-sync
#   /credentials/<provider>/.domains      — domain hint (one host per line),
#                                           written by gate when registering
#                                           the provider so the sidecar knows
#                                           which domains to seed cookies on.
#
# Other creds:
#   /credentials/google/{client.json,tokens.json} — google OAuth
#   /credentials/mcp/oauth/<hash>/tokens.json     — notion etc. MCP OAuth

ENTRYPOINT ["/usr/local/bin/spwn-gate"]
CMD ["run"]
