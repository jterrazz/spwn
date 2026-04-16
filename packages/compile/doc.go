// Package compile is the spwn compiler: provider-neutral source
// (spwn.yaml + spwn/agents/* + skills + hooks) → runtime-specific
// file layout a concrete agent runtime can boot from.
//
// Phase 1, Compile(name, input), is a pure function: given an
// Input it returns a *Tree, a deterministic path→bytes map. No
// disk writes, no Docker. Same input → same bytes → golden tests
// diff cleanly.
//
// Phase 2, materialisation, lives in packages/architect (spawn-time
// docker-cp into the running container) or in packages/image
// (build-time COPY into a derived image). Compile is deliberately
// oblivious to which delivery shape consumes its output.
//
// Runtime-specific renderers live in packages/compile/runtimes/
// (claude_code today). Adding a new runtime is a sub-package with
// a Render method and an init() that calls Register.
package compile
