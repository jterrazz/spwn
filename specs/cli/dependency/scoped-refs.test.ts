import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * The install/uninstall specs whose subject is a COUNT or an ABSENCE in a
 * manifest: exactly one list entry after a repeated install, no entry left
 * after an uninstall, the three agents a scoped install must not have
 * touched. `files:` says what a file contains, never how many times and
 * never what it does not, so each session stays in its own document —
 * asserted whole by `cli.run()`, every banner and every exit code — and
 * the code adds the one probe the format cannot state.
 *
 * Every other install and uninstall spec is a document beside this file.
 */

const AGENTS = ['mark', 'helly', 'irving', 'dylan'] as const;

/** List entries naming `ref` in a manifest — comment mentions do not count. */
function entriesOf(manifest: string, ref: string): string[] {
    return manifest.match(new RegExp(String.raw`^\s*-\s+["']?${ref}["']?\s*$`, 'gm')) ?? [];
}

test('a repeated install leaves exactly one manifest entry', async () => {
    // Given - the same ref installed twice, stated in the document
    const result = await cli.run('repeated-install.spec.yaml');

    // Then - the manifest carries one list entry, not two
    expect(
        entriesOf(result.file('spwn/agents/neo/agent.yaml').content, 'spwn:python'),
    ).toHaveLength(1);
});

test('a project-wide install over a partly-installed project duplicates nothing', async () => {
    // Given - mark already carries the ref when the project-wide install runs
    const result = await cli.run('mixed-state-install.spec.yaml');

    // Then - every agent ends with exactly one entry, mark included
    for (const name of AGENTS) {
        const entries = entriesOf(
            result.file(`spwn/agents/${name}/agent.yaml`).content,
            'spwn:qmd',
        );
        expect(entries, `${name} should carry exactly one spwn:qmd entry`).toHaveLength(1);
    }
});

test('a scoped install leaves the other agents untouched', async () => {
    // Given - the ref installed against mark only
    const result = await cli.run('scoped-install.spec.yaml');

    // Then - the other three manifests never mention it
    for (const name of ['helly', 'irving', 'dylan']) {
        expect(
            result.file(`spwn/agents/${name}/agent.yaml`).content,
            `${name} should not carry spwn:qmd`,
        ).not.toContain('spwn:qmd');
    }
});

test('a round-trip uninstall leaves neither the manifest nor the lockfile carrying it', async () => {
    // Given - a ref installed then immediately uninstalled
    const result = await cli.run('round-trip-uninstall.spec.yaml');

    // Then - the manifest drops the ref, and any surviving lockfile drops the pin
    expect(result.file('spwn/agents/neo/agent.yaml').content).not.toContain('spwn:python');
    const lock = result.file('spwn.lock');
    if (lock.exists) {
        expect(lock.content).not.toContain('spwn:python');
    }
});

test('a scoped uninstall takes the ref from that agent only', async () => {
    // Given - all four agents carry the ref, then mark loses it
    const result = await cli.run('scoped-uninstall.spec.yaml');

    // Then - mark is clean; the document already proved the other three kept it
    expect(result.file('spwn/agents/mark/agent.yaml').content).not.toContain('spwn:qmd');
});

test('the last carrier losing a ref clears it everywhere', async () => {
    // Given - only mark carried the ref, then mark loses it
    const result = await cli.run('last-carrier-uninstall.spec.yaml');

    // Then - no agent carries it and the lockfile no longer pins it
    for (const name of AGENTS) {
        expect(
            result.file(`spwn/agents/${name}/agent.yaml`).content,
            `${name} should not carry spwn:qmd`,
        ).not.toContain('spwn:qmd');
    }
    const lock = result.file('spwn.lock');
    if (lock.exists) {
        expect(lock.content).not.toContain('spwn:qmd');
    }
});

test('a bare-name uninstall removes the scheme form it added', async () => {
    // Given - a ref installed and uninstalled both by its bare name
    const result = await cli.run('bare-name-uninstall.spec.yaml');

    // Then - the resolver is present on the uninstall path too
    expect(result.file('spwn/agents/neo/agent.yaml').content).not.toContain('spwn:python');
});
