# outside: the gate image runs three things — the spwn-gate Go binary (the entrypoint: cookie sync and MCP routing
# on :9000), the gate-browser sidecar it supervises (a Playwright Chromium pool on 127.0.0.1:9001), and the catalog
# tools it spawns, which require @spwn/gate-tool. packages/gate/lifecycle.go stages the context: the linux binary at
# ./spwn-gate, ./browser and ./sdk (each its package.json and src/), and this file.
FROM mcr.microsoft.com/playwright:v1.59.1-jammy

# outside: the base image ships Chromium but not the playwright npm module; jq inspects cookies, gws drives the
# gmail/gcal elements, and the docker CLI lets spwn:dispatch run `spwn agent talk` over the bind-mounted socket.
RUN apt-get update && apt-get install -y --no-install-recommends jq docker.io \
 && rm -rf /var/lib/apt/lists/* \
 && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm install -g playwright @googleworkspace/cli

ENV NODE_PATH=/usr/lib/node_modules

RUN groupadd --system spwn && useradd --system --gid spwn --create-home --shell /bin/bash spwn

COPY spwn-gate /usr/local/bin/spwn-gate
COPY browser /opt/gate-browser
COPY sdk /opt/gate-tool
# invariant: Node runs the sidecar and the SDK from their TypeScript sources, and it strips types only outside
# node_modules — so the SDK lives at /opt/gate-tool and a catalog tool's require('@spwn/gate-tool') reaches it through
# the symlink NODE_PATH resolves, and the sidecar's `import 'playwright'` reaches the global install through its own.
RUN mkdir -p /usr/lib/node_modules/@spwn \
 && ln -s /opt/gate-tool /usr/lib/node_modules/@spwn/gate-tool \
 && ln -s /usr/lib/node_modules /opt/gate-browser/node_modules \
 && chmod +x /usr/local/bin/spwn-gate

USER spwn
WORKDIR /home/spwn

EXPOSE 9000

ENTRYPOINT ["/usr/local/bin/spwn-gate"]
CMD ["run"]
