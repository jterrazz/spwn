# spwn-gate — host-side credential broker / element bridge.
#
# Two layers:
#
#   1. Reverse-proxy elements (notion, future linear/atlassian/…):
#      pure Go, the binary handles them entirely.
#
#   2. Backend-CLI elements (gmail, gcal, …): the gate hand-rolls a
#      minimal MCP HTTP server and translates tools/call to subprocess
#      invocations of gws (Google Workspace CLI). Requires Node.js
#      runtime + the npm-installed gws binary.
#
# The Go binary is cross-compiled on the host (so go.work + every
# workspace module resolve correctly) and COPY'd in. Same pattern as
# packages/architect builds its image.
#
# Build context expected: a directory containing the linux binary at
# ./spwn-gate. packages/gate/lifecycle.go produces that.

FROM node:20-alpine

# Install gws once, baked into the image. Avoids an `npm install` on
# every container start.
RUN npm install -g @googleworkspace/cli && \
    addgroup -S spwn && adduser -S spwn -G spwn

COPY spwn-gate /usr/local/bin/spwn-gate
RUN chmod +x /usr/local/bin/spwn-gate

USER spwn
WORKDIR /home/spwn

EXPOSE 9000

# Bind mounts (runtime, set by spwn gate start):
#   /credentials  ← ~/.spwn/credentials  (ro)  — MCP OAuth tokens for proxy elements
#   /gate         ← ~/.spwn/gate         (rw)  — persistent state, gws creds
#
# gws looks for credentials at $GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE,
# which the gate sets per-request to /gate/google/credentials.json.

ENTRYPOINT ["/usr/local/bin/spwn-gate"]
CMD ["run"]
