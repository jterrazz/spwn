import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Grammar-contract suite for dependency refs. The CLI accepts bare names and
 * auto-promotes them to spwn:<name>; manifests stay strict. These are scalpel
 * probes into the grammar wording and on-disk manifest shape — the full report
 * goldens live in check.test.ts.
 */

test('spwn check rejects bare / local: / @owner refs in agent.yaml', async () => {
    // Given - check-legacy-refs mixes three rejected shapes: bare, local:, @owner
    await using result = await cli.fixture('$FIXTURES/check-legacy-refs/').exec('check');

    // Then - each rejected ref surfaces its own error and every hint names the grammar
    expect(result.exitCode).not.toBe(0);
    expect(result.stdout).toContain('dependency "python" is invalid');
    expect(result.stdout).toContain('dependency "local:foo" is invalid');
    expect(result.stdout).toContain('dependency "@acme/foo" is invalid');
    expect(result.stdout).toContain('skill/<name>');
    expect(result.stdout).toContain('tool/<name>');
    expect(result.stdout).toContain('hook/<name>');
    expect(result.stdout).toContain('spwn:<name>');
    expect(result.stdout).toContain('github:<owner>/<repo>');
});

test('bare names accepted on the cli land as spwn:<name> in agent.yaml', async () => {
    // Given - an empty project; the resolver promotes bare "python" before it reaches disk
    await using result = await cli
        .fixture('$FIXTURES/empty/')
        .exec(['init', 'install python', 'check']);

    // Then - check passes and the manifest carries only the canonical form
    expect(result.exitCode, result.stderr.text).toBe(0);
    const manifest = result.file('spwn/agents/neo/agent.yaml').content;
    expect(manifest).toContain('spwn:python');
    expect(manifest).not.toMatch(/^\s*-\s+python\s*$/m);
});

test('manifest is the strict boundary even when the cli would accept', async () => {
    // Given - an init-scaffolded project run through installs mixing bare, local, and catalog refs
    await using result = await cli
        .fixture('$FIXTURES/empty/')
        .exec(['init', 'install qmd', 'install skill/focus --agent neo', 'install spwn:unix']);

    // Then - every entry on disk carries an explicit scheme; nothing is bare
    expect(result.exitCode, result.stderr.text).toBe(0);
    const manifest = result.file('spwn/agents/neo/agent.yaml').content;
    expect(manifest).toContain('spwn:qmd');
    expect(manifest).toContain('skill/focus');
    expect(manifest).toContain('spwn:unix');
    const bareHits = manifest.match(/^\s*-\s+[a-z0-9][a-z0-9-]*\s*$/gm);
    expect(bareHits, `manifest carries bare entries: ${bareHits?.join(', ')}`).toBeNull();
});
