import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Tool-reference classification by `spwn check`. These pin the classifier edges
 * as presence/absence probes; the full registry-error golden lives in check.test.ts.
 */

test('accepts mixed spwn: and tool/ refs without error', async () => {
    // Given - mixed-tool-refs declares spwn:unix plus a local tool/ that exists on disk
    await using result = await cli.fixture('$FIXTURES/mixed-tool-refs/').exec('check');

    // Then - check passes; neither the registry nor "does not exist" fires
    expect(result.exitCode, result.stdout.text).toBe(0);
    expect(result.stdout).not.toContain('remote registries are not yet supported');
    expect(result.stdout).not.toContain('does not exist');
});

test('rejects github registry refs with explicit wording', async () => {
    // Given - check-registry-tool declares dependencies: ["github:jterrazz/foo"]
    await using result = await cli.fixture('$FIXTURES/check-registry-tool/').exec('check');

    // Then - exits non-zero and the error quotes the offending ref
    expect(result.exitCode).toBe(1);
    expect(result.stdout).toContain('remote registries are not yet supported');
    expect(result.stdout).toContain('github:jterrazz/foo');
});
