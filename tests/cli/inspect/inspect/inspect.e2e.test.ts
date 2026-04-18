import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn inspect` prints a kubectl-describe / cargo-tree blend: a
 * per-agent block with an identity header, a resolved dependency tree
 * (with (*)-dedup + composition badges), a skills list, and a hooks
 * list.
 *
 * Tests use --offline so the output is fully deterministic — no live
 * world-status lookup, every agent renders with "○ stopped".
 */

describe('spwn inspect', () => {
    test('renders a single-agent project end-to-end', async () => {
        // Given - the frozen single-agent fixture (one agent, three spwn: deps)
        const result = await spec('inspect single')
            .project('single-agent')
            .exec('inspect --offline')
            .run();

        // Then - exits clean and the block matches the golden fixture
        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        await result.stdout.toMatch('single-agent.txt');
    });

    test('focuses on a single named agent', async () => {
        const result = await spec('inspect named')
            .project('single-agent')
            .exec('inspect neo --offline')
            .run();

        expect(result.exitCode).toBe(0);
        // One block only — no second "Name" header (no trailing agents).
        const headerCount = (result.stdout.text.match(/(^|\n)Name\s+/g) ?? []).length;
        expect(headerCount).toBe(1);
        expect(result.stdout.text).toContain('Name         neo');
    });

    test('errors cleanly when the named agent is missing', async () => {
        const result = await spec('inspect missing')
            .project('single-agent')
            .exec('inspect ghost --offline')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('agent "ghost" not found');
    });

    test('errors when run outside a spwn project', async () => {
        const result = await spec('inspect no-project')
            .project('empty')
            .exec('inspect --offline')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('no spwn.yaml found');
    });

    test('renders a local tool dependency without the local: prefix', async () => {
        // Given - mixed-tool-refs declares `tool:my-local-tool`. The
        // Local tool directory exists under spwn/tools/ with no
        // Tool.yaml, so inspect should fall back gracefully.
        const result = await spec('inspect local')
            .project('mixed-tool-refs')
            .exec('inspect --offline')
            .run();

        expect(result.exitCode).toBe(0);
        // Local tool shows as bare name, not "local:my-local-tool".
        expect(result.stdout.text).toContain('my-local-tool');
        expect(result.stdout.text).not.toContain('local:my-local-tool');
    });

    test('--help prints the inspect command usage', async () => {
        const result = await spec('inspect help').project('empty').exec('inspect --help').run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('inspect');
        expect(result.stdout.text).toContain('--offline');
    });
});
