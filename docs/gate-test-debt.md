# Gate refactor — test debt checklist

Snapshot of where we stand after the refactor that moved external services
(X today, LinkedIn / Reddit tomorrow) from hardcoded `packages/gate/` code
to a generic catalog-tool runtime. Reference for what's covered, what's
deferred, and what's intentionally out of scope.

Last reviewed: **2026-04-25**.

## Layer 1 — Gate primitives (Go core)

### `packages/gate/sidecar.go` — Playwright supervisor

| | Test | Status |
|--|--|--|
| ✅ | Sidecar boots + answers `/healthz` (manual QA) | covered |
| ✅ | Sidecar restarts after SIGKILL (manual QA-1) | covered |
| ⏳ | Backoff doubles on rapid crash (1s→2s→4s…→30s cap) | unit test deferred |
| ⏳ | Backoff resets to 1s after a clean exit | unit test deferred |
| ⏳ | `prefixWriter` splits multi-line stdout into prefixed log lines | unit test deferred |
| ⏳ | `Run` exits cleanly on `ctx.Done()` mid-spawn | unit test deferred |

### `packages/gate/toolproc.go` — ToolSupervisor + ToolElement

| | Test | Status |
|--|--|--|
| ✅ | Allocates ports sequentially from `BaseToolPort` | `tools_test.go::TestNewToolSupervisor_AllocatesPortsSequentially` |
| ✅ | Skips tools without `MCP.Entry` (cookies-only) | `TestNewToolSupervisor_SkipsToolsWithoutMCPEntry` |
| ✅ | Returns 503 with `not ready` JSON when upstream is down | `TestToolSupervisor_503WhenUpstreamDown` |
| ✅ | `Refresh` is a no-op (token refresh owned by subprocess) | `TestToolElement_RefreshIsNoOp` |
| ✅ | Tool subprocess restarts after SIGKILL (manual QA-2) | covered |
| ⏳ | `superviseOne` exits when `ctx.Done()` fires mid-Wait | unit test deferred |
| ⏳ | Env vars (`GATE_TOOL_NAME`, `GATE_TOOL_PORT`, `GATE_BROWSER_URL`, `GATE_CREDENTIALS_DIR`) reach the subprocess | integration test deferred |
| ⏳ | Reverse proxy strips the `/mcp/<name>/` prefix before forwarding | unit test deferred |
| ⏳ | Wait-healthy polls /healthz every 300ms, gives up at 15s | unit test deferred |

### `packages/gate/tools.go` — catalog tool loader

| | Test | Status |
|--|--|--|
| ✅ | Picks up only tools with a `gate:` section | `tools_test.go::TestLoadTools_PicksUpGateShapedTools` |
| ✅ | Sorts results alphabetically | `TestLoadTools_SortsByName` |
| ✅ | Missing root dir returns empty (no error) | `TestLoadTools_MissingDirReturnsEmpty` |
| ✅ | Malformed YAML surfaces a clear error | `TestLoadTools_MalformedYAMLReturnsError` |
| ✅ | Tools without `gate:` section silently skipped | `TestLoadTools_NoGateSectionSilentlySkipped` |
| ✅ | `CookieProvider()` returns nil when spec has no cookies | `TestTool_CookieProviderNilWhenSpecEmpty` |
| ⏳ | Subdir without `tool.yaml` skipped | implicit in existing tests |

### `packages/gate/browser_element.go` — generic /mcp/browser

| | Test | Status |
|--|--|--|
| ✅ | All 10 tools registered (open/close/goto/click/type/scroll/wait-selector/wait-response/captured-responses/eval) | `browser_element_test.go::TestBrowserElement_RegistersTenTools` |
| ✅ | Forwards session ops to `/sessions/<id>/<op>` with correct method + body | `TestBrowserElement_ForwardsToSessionEndpoints` |
| ✅ | `browser-open` POSTs to `/sessions` and returns the id | `TestBrowserElement_OpenSessionRequiresProvider` |
| ✅ | Missing `session_id` returns a clear error | `TestBrowserElement_ForwardSessionMissingIDFails` |
| ✅ | `sessionSchema` produces valid JSON schema with required fields | `TestSessionSchema_BuildsValidJSON` |
| ⏳ | End-to-end MCP call (`/mcp/browser` → sidecar) — covered by manual QA-6 only | E2E deferred |

### `packages/gate/cookies.go` — `WriteDomainHints`

| | Test | Status |
|--|--|--|
| ✅ | Writes one `.domains` file per provider with full domain list | `cookies_domains_test.go::TestWriteDomainHints_WritesOneFilePerProvider` |
| ✅ | Empty registry is a no-op (no spurious dirs) | `TestWriteDomainHints_EmptyRegistry_NoOp` |
| ✅ | File perms = 0600 (matches cookies.json) | `TestWriteDomainHints_FilePermsRestrictive` |

## Layer 2 — Node sidecar + SDK (apps/gate/browser, apps/gate/sdk)

| | Test | Status |
|--|--|--|
| ✅ | Sidecar HTTP API works against real Playwright (manual QA-1, QA-4, QA-5) | E2E covered |
| ✅ | Session reaper drops sessions after 5 min idle | implicit (manual; not asserted) |
| ⏳ | Node tests for SDK `Tool` class (manifest dump, MCP server, CLI dispatch) | not started |
| ⏳ | Node tests for `Session` client (request shape, error propagation) | not started |
| ⏳ | Sidecar handles 100 concurrent sessions without leaking | load test deferred |
| ⏳ | Sidecar refuses non-loopback requests (defense-in-depth) | currently relies on bind to 127.0.0.1 |

**Note:** No JS test runner is wired into the repo today. Adding `vitest` or `node:test` for the sidecar+SDK is its own piece of yak-shaving — defer until a Node-side bug actually bites.

## Layer 3 — `catalog/x/`

| | Test | Status |
|--|--|--|
| ✅ | `mcp-manifest` returns 7 methods (manual QA: `node index.js mcp-manifest`) | covered |
| ✅ | `fetch-home`, `fetch-account`, `search`, `fetch-favorites`, `fetch-thread` end-to-end (manual QA-8) | covered |
| ✅ | `post-tweet` end-to-end (manual QA-8c posted real tweet) | covered |
| ⏳ | `reply-tweet` end-to-end after the post-tweet test | covered earlier session, not re-run today |
| ⏳ | Field extraction (is_retweet, quoted, retweeted, views, media) on a known-shape fixture | unit test deferred — needs golden JSON |
| ⏳ | Scroll cap kicks in at MAX_SCROLLS (60) | not asserted |
| ⏳ | End-of-feed detection trips after STALE_SCROLLS_FOR_END (3) | not asserted |

## Phase 4 — Agent policy

### `packages/agent/internal/manifest/schema.go`

| | Test | Status |
|--|--|--|
| ✅ | Mixed scalar + mapping deps parse | `policy_test.go::TestManifest_UnmarshalYAML_PoliciedDeps` |
| ✅ | Mapping without `name:` rejected | `TestManifest_UnmarshalYAML_MissingNameRejected` |
| ✅ | Non-scalar/non-mapping entry rejected | `TestManifest_UnmarshalYAML_BadTypeRejected` |
| ✅ | Marshal round-trips with policy intact | `TestManifest_RoundtripPreservesPolicy` |
| ✅ | Empty `{name: ...}` mapping doesn't materialize a DepPolicies entry | `TestManifest_EmptyDepPolicy_NotMaterialized` |
| ✅ | Real brain agent.yamls parse + policies extracted (manual QA-9) | covered |

### `apps/cli/world/project.go::mergeDepPolicy`

| | Test | Status |
|--|--|--|
| ✅ | Deny union across two policies | `policy_merge_test.go::TestMergeDepPolicy_DenyUnion` |
| ✅ | Allow intersection across two policies | `TestMergeDepPolicy_AllowIntersection` |
| ✅ | Empty side passes the other through | `TestMergeDepPolicy_EmptyPassesOtherThrough` |
| ✅ | Conflict (one allow, one deny) — deny wins | `TestMergeDepPolicy_OneSideAllowOtherDeny` |

### `packages/compile/internal/dockerfile/generator.go` — Policy emission

| | Test | Status |
|--|--|--|
| ✅ | Non-empty Policy emits `RUN ... > /etc/spwn/policy/<short>.json` | `policy_test.go::TestGenerate_EmitsPolicyJSONWhenSet` |
| ✅ | Empty Policy emits no RUN | `TestGenerate_OmitsPolicyWhenEmpty` |
| ✅ | Nil Policy emits no RUN | `TestGenerate_OmitsPolicyWhenNil` |
| ✅ | Single-quote in method name doesn't break shell escaping | `TestGenerate_EscapesSingleQuotesInPolicy` |
| ✅ | Policy RUN appears AFTER tool install commands (so wrapper is in place when tested) | `TestGenerate_PolicyEmittedAfterToolCommands` |

### `catalog/mcp2cli/.../tool.yaml` — `spwn-policy-check`

| | Test | Status |
|--|--|--|
| ✅ | No policy file → allow all (manual QA-7 test 1) | covered |
| ✅ | Deny excludes method → allow (test 2) | covered |
| ✅ | Deny includes method → reject (test 3) | covered |
| ✅ | Allow positive list, miss → reject (test 4) | covered |
| ✅ | Allow positive list, hit → allow (test 5) | covered |
| ✅ | `tools/list` always allowed (test 6) | covered |
| ✅ | Malformed policy json → fail-open (test 7) | covered |
| ⏳ | Policy file inside actual built world image (E2E) | deferred — needs `spwn build` + `docker run` |

## Phase 5 — Browser extension

All `apps/spwn-cookie-sync/` testing is **manual only** — the extension
needs real Chrome + a user gesture for `chrome.permissions.request`.

| | Test | Status |
|--|--|--|
| ✅ | Per-provider domain hint file picked up by sidecar (proven indirectly by QA-1) | covered |
| ⏳ | Popup "Grant access" button triggers MV3 permission prompt | manual only |
| ⏳ | After grant, cookies start flowing within 5min refresh window | manual only |
| ⏳ | Provider list reflects newly-installed catalog tools after gate restart | manual only |

## Phase 6 — `/mcp/browser` element

| | Test | Status |
|--|--|--|
| ✅ | Open / goto / eval / close end-to-end (manual QA-6) | covered |
| ✅ | Unit-level forwarding shape | `browser_element_test.go` |
| ⏳ | Each of the 10 tools wired correctly (only goto/eval/close exercised live) | deferred |
| ⏳ | wait-response timeout returns clear error | deferred |

## Phase 7 — Brain wiring

| | Test | Status |
|--|--|--|
| ✅ | Brain agent.yamls parse with new schema (manual QA-9) | covered |
| ✅ | publish.sh fails fast when gate tool not installed (read by inspection) | covered by code review |
| ⏳ | publish.sh end-to-end with a real approved draft → live tweet | manual only, deferred until next real publish |

## Cross-cutting / system

| | Test | Status |
|--|--|--|
| ✅ | Full Go test suite green (manual QA-10) | covered |
| ✅ | `spwn install spwn:x` materialises `~/.spwn/gate/tools/x/` (verified end-to-end) | covered |
| ⏳ | `spwn uninstall spwn:x` cleans up `~/.spwn/gate/tools/x/` | NOT IMPLEMENTED — known gap |
| ⏳ | `spwn install spwn:x` after a catalog code change refreshes the tool dir | should work (overwrites), not asserted |
| ⏳ | Concurrent `spwn install` runs don't corrupt the tool dir | unlikely in practice; not asserted |
| ⏳ | Image hash includes per-agent policy bytes (so policy changes invalidate cache) | review needed |

## Highest-risk untested paths

1. **`spwn uninstall` doesn't clean gate-tools** — installing then uninstalling leaves the tool live in the gate. Fix is a one-liner (mirror the install hook). Not done.
2. **Image cache may not invalidate on policy-only change** — `hashBuildContext` hashes the Dockerfile bytes; the policy RUN is in those bytes, so it _should_ invalidate. Verify by inspection or a focused test.
3. **End-to-end policy enforcement inside an actual built world container** — only `spwn-policy-check` is unit-tested standalone. The full path (build image with policy → run agent → call denied method → reject) hasn't been exercised.

## Out of scope (deliberately)

- Node-side test runner (vitest/node:test) for sidecar + SDK — defer until a JS bug bites.
- Property-based tests for the YAML unmarshaller — current explicit cases cover the realistic shapes.
- Performance / load tests on the sidecar — current usage is single-user, sequential. Revisit when concurrent agents hit the same tool.
- Multi-agent worlds with conflicting policies — `mergeDepPolicy` covers it logically; real-world test deferred until a second-agent world ships.
