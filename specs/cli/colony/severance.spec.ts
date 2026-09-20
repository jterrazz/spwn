import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Severance catalog full-cycle smoke test. The severance entry is the
 * canonical 4-agent showcase (Mark + Helly/Irving/Dylan in the `mdr`
 * world). A fast CLI-only pass proves install + check + on-disk layout;
 * a docker-gated pass spins the full 4-agent colony up and verifies the
 * roster. Every result binds with `await using` (rule B5).
 */
describe('severance catalog', () => {
    test('severance agents carry distinct SOUL.md content', async () => {
        // Given - the severance gallery installed into an empty dir
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init severance');

        // Then - every pair of agent souls is byte-different (no shared voice)
        expect(result.exitCode).toBe(0);
        const souls = ['mark', 'helly', 'irving', 'dylan'].map(
            (name) => result.file(`spwn/agents/${name}/SOUL.md`).content,
        );
        for (let i = 0; i < souls.length; i += 1) {
            for (let j = i + 1; j < souls.length; j += 1) {
                expect(souls[i], `souls ${i} and ${j} are byte-identical`).not.toBe(souls[j]);
            }
        }
    });

    test('mdr world spawns all four agents in one container', async () => {
        // Given - the severance gallery installed then brought up as the mdr world
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'up mdr']);

        // Then - the mdr container is running and its roster inlines every MDR member
        expect(result.exitCode, result.stderr.text).toBe(0);
        const mdr = result.container('mdr');
        expect(mdr.running).toBe(true);
        const claude = mdr.file('/agents/mark/CLAUDE.md').content;
        for (const name of ['mark', 'helly', 'irving', 'dylan']) {
            expect(claude, `roster missing ${name}`).toContain(name);
        }
    });
});
