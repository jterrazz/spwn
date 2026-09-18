import { specification } from '@jterrazz/test';
import { resolve } from 'node:path';
import { afterAll } from 'vitest';

/**
 * CLI specification runner for spwn — the single product runner.
 *
 * The spwn binary IS the product: every spec exercises the compiled
 * binary directly, never a tool underneath it. Each spec runs in a fresh
 * temp cwd; `.fixture('$FIXTURES/<project>/')` spreads a committed project
 * tree into it, `.env()` isolates the child environment.
 *
 * Docker-aware mode is opt-in via the `docker` option. The contract
 * with the binary: it labels every container it spawns with
 * `sh.spwn.test.run = <value of SPWN_TEST_LABEL>` (see
 * packages/world/internal/labels). The runner injects a per-run id into
 * SPWN_TEST_LABEL, finds containers by that label, and force-removes
 * them at scope exit — which is why every docker-aware result MUST bind
 * with `await using` (rule B5). Container lookup keys off
 * `sh.spwn.world.config` (the manifest world key) because spwn only
 * sets `sh.spwn.world.name` when the user assigns an explicit display
 * name — empty for fixture-declared worlds.
 *
 * The `env` record names the environment sets a `<case>.spec.yaml`
 * document reaches for by bare word (`env: isolated`). A document
 * states WHICH ground it stands on; how that ground is built stays
 * here, once. `isolated` moves the user's spwn home onto the spec's
 * temp cwd — without it a spec reads, and writes, the operator's real
 * ~/.spwn: its agents, its activity log, its credentials. Keep the
 * record free of comments: the conventions checker reads these literals
 * by scan, and a block comment between them hides the keys a document
 * is allowed to name.
 */
const SPWN_BIN = resolve(import.meta.dirname, '../../.artifacts/go/spwn');

/*
 * Blanket-disable live credential validation in the test process so child
 * spwn invocations never hit real provider APIs. The auth dashboard and the
 * spawn pre-flight both run validations; a parallel e2e suite hitting
 * Anthropic /oauth/usage burns rate limits and flakes on 429s. A spec that
 * wants to exercise validation opts back in with
 * `.env({ SPWN_SKIP_AUTH_VALIDATION: null })`.
 */
process.env.SPWN_SKIP_AUTH_VALIDATION = '1';

export const { cli, cleanup } = await specification.cli(SPWN_BIN, {
    docker: {
        envVar: 'SPWN_TEST_LABEL',
        nameLabel: 'sh.spwn.world.config',
        testRunLabel: 'sh.spwn.test.run',
    },
    env: {
        isolated: { SPWN_HOME: '$WORKDIR/spwn-home' },
    },
});

afterAll(cleanup);
