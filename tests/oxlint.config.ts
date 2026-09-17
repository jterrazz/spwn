import { testing } from '@jterrazz/test/oxlint';
import { compose, defineConfig, node } from '@jterrazz/typescript/oxlint';

/*
 * Node preset + the @jterrazz/test conventions plugin. Every tree this member
 * owns is source — the CLI specs, the Playwright suite, the smoke test, the
 * contract asserter. What is not source, `_fixtures/` inputs and `_expected/`
 * goldens, the profile already ignores; the Go catalog tests and the shell
 * simulators hold nothing oxlint reads.
 */
export default defineConfig(compose(node, testing));
