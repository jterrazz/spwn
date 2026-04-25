# spwn-gate — host-side credential broker / element bridge.
#
# The binary is cross-compiled on the host (so go.work + every module
# in the workspace resolve correctly) and COPY'd in. Same pattern as
# the architect image — packages/architect/build.go cross-compiles
# spwn for linux/<arch> the same way.
#
# Build context expected: a directory containing the pre-compiled
# linux binary at ./spwn-gate. The lifecycle helper in
# packages/gate/lifecycle.go produces that.

FROM gcr.io/distroless/static-debian12:nonroot

COPY spwn-gate /spwn-gate

EXPOSE 9000

# Bind mounts (runtime, set by spwn gate start):
#   /credentials  ← ~/.spwn/credentials  (ro)
#   /gate         ← ~/.spwn/gate         (rw, persistent state)

ENTRYPOINT ["/spwn-gate"]
CMD ["run"]
