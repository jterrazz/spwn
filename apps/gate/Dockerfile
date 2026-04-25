# spwn-gate — host-side credential broker / element bridge.
#
# Two layers of elements:
#
#   1. Reverse-proxy elements (notion, future linear/atlassian/…):
#      pure Go, the binary handles them entirely.
#
#   2. Backend-CLI elements: the gate hand-rolls a minimal MCP HTTP
#      server (packages/gate/mcpserver.go) and translates tools/call
#      to subprocess invocations of a backend CLI:
#        - gmail / gcal → gws (Google Workspace CLI, npm)
#        - x            → twscrape via x-fetch helper (pip)
#
# The Go binary is cross-compiled on the host (so go.work + every
# workspace module resolve correctly) and COPY'd in. Same pattern as
# packages/architect builds its image.
#
# Build context expected: a directory containing the linux binary at
# ./spwn-gate. packages/gate/lifecycle.go produces that.

FROM node:20-alpine

# System deps: python3 for twscrape, jq for the x-fetch helper to read
# cookies.json without pulling in a JSON-parsing Python dep.
RUN apk add --no-cache python3 py3-pip jq && \
    addgroup -S spwn && adduser -S spwn -G spwn

# gws (Google Workspace CLI) — used by the gmail/gcal elements.
RUN npm install -g @googleworkspace/cli

# twscrape — used by the x element. Cookie auth, well-maintained
# unofficial X scraper. Read-only by design (search, profiles,
# tweets, bookmarks, followers); writes flow through agent-browser
# in publish.sh.
RUN pip3 install --break-system-packages twscrape

COPY spwn-gate /usr/local/bin/spwn-gate
COPY x-fetch /usr/local/bin/x-fetch
RUN chmod +x /usr/local/bin/spwn-gate /usr/local/bin/x-fetch

USER spwn
WORKDIR /home/spwn

EXPOSE 9000

# Bind mounts (runtime, set by spwn gate start):
#   /credentials  ← ~/.spwn/credentials  (rw)  — gate is the cred broker
#   /gate         ← ~/.spwn/gate         (rw)  — persistent state
#
# Cookie sources (browser extension writes; elements read):
#   /credentials/x/cookies.json        — auth_token + ct0 (twscrape)
#   /credentials/linkedin/cookies.json — li_at + JSESSIONID (future)
#
# Other creds:
#   /credentials/google/{client.json,tokens.json} — google OAuth
#   /credentials/mcp/oauth/<hash>/tokens.json     — notion etc. MCP OAuth
#   /gate/cookie-sync-secret                       — extension pairing secret

ENTRYPOINT ["/usr/local/bin/spwn-gate"]
CMD ["run"]
