import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, describe, expect, test } from 'vitest';

import { MindAssertion } from '../../setup/mind-assertion.js';
import {
    expectLine,
    expectNoLine,
    expectTableHeader,
    expectTableRow,
    stripAnsi,
} from '../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../setup/spwn.specification.js';

/**
 * FULL AGENT LIFECYCLE E2E TEST
 *
 * Tests the complete user journey end-to-end:
 *   init → agent ls → up (Docker) → ls → inspect → logs → down →
 *   dream → sleep → fork → export → rm → verify cleanup
 */
describe('full agent lifecycle', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test('complete user journey: init → up → evolve → fork → cleanup', () => {
        // ── STEP 1: spwn init ──────────────────────────────────────
        // CreateTestContext already scaffolds a `neo` agent on disk.
        ctx = createTestContext();
        const initResult = ctx.spwn(['init']);
        expect(initResult.exitCode).toBe(0);
        expectLine(initResult.output, /✓ Created config\s+\w+\.yaml/);

        // ── STEP 2: agent ls ───────────────────────────────────────
        const lsResult1 = ctx.spwn(['agent', 'ls']);
        expect(lsResult1.exitCode).toBe(0);
        expectTableHeader(lsResult1.output, ['NAME', 'WORLD', 'STATUS']);
        expectTableRow(lsResult1.output, ['neo']);

        // ── STEP 3: spwn up --agent neo -w <workspace> (Docker) ────
        const spawnResult = ctx.spwn(['up', '--agent', 'neo', '-w', ctx.home], 60_000);
        expect(spawnResult.exitCode).toBe(0);
        const worldId = parseWorldId(spawnResult.output);
        expect(worldId).toBeTruthy();
        expectLine(spawnResult.output, /✓ Created container\s+(?:spwn-world|w)-\w+-\d{5}/);

        // ── STEP 4: spwn ls - verify world appears ─────────────────
        const lsWorldsResult = ctx.spwn(['ls']);
        expect(lsWorldsResult.exitCode).toBe(0);
        expect(stripAnsi(lsWorldsResult.output)).toContain(worldId!);
        expectTableHeader(lsWorldsResult.output, ['ID', 'CONFIG', 'AGENTS', 'STATUS']);

        // ── STEP 5: spwn world inspect <id> - verify details ───────
        const inspectWorldResult = ctx.spwn(['world', 'inspect', worldId!]);
        expect(inspectWorldResult.exitCode).toBe(0);
        expectLine(
            inspectWorldResult.output,
            new RegExp(`World:\\s+${worldId!.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)}`),
        );
        expectLine(inspectWorldResult.output, /Status:\s+(running|idle)/);
        expectLine(inspectWorldResult.output, /Config:\s+default/);

        // ── STEP 6: spwn world logs <id> - verify it doesn't crash ─
        const logsResult = ctx.spwn(['world', 'logs', worldId!]);
        expect(logsResult.exitCode).toBe(0);
        expect(typeof logsResult.output).toBe('string');

        // ── STEP 7: spwn down <id> - destroy world ─────────────────
        const downResult = ctx.spwn(['down', worldId!], 30_000);
        expect(downResult.exitCode).toBe(0);
        expectLine(downResult.output, /✓ World destroyed\. Agent survives\./);

        // Verify world is gone from ls
        const lsAfterDown = ctx.spwn(['ls']);
        expect(lsAfterDown.exitCode).toBe(0);
        expectNoLine(
            lsAfterDown.output,
            new RegExp(worldId!.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)),
        );

        // ── STEP 8: spwn agent dream neo ───────────────────────────
        const reflectResult = ctx.spwn(['agent', 'dream', 'neo']);
        expect(reflectResult.exitCode).toBe(0);
        expectLine(reflectResult.output, /→ Dreaming for agent "neo"\.\.\./);

        // ── STEP 9: spwn agent sleep neo ───────────────────────────
        const sleepResult = ctx.spwn(['agent', 'sleep', 'neo']);
        expect(sleepResult.exitCode).toBe(0);
        expectLine(sleepResult.output, /→ Sleep cycle for agent "neo"\.\.\./);

        // ── STEP 10: spwn agent fork neo neo-v2 ────────────────────
        const forkResult = ctx.spwn(['agent', 'fork', 'neo', 'neo-v2']);
        expect(forkResult.exitCode).toBe(0);
        expectLine(forkResult.output, /→ Forking "neo" -> "neo-v2"\.\.\./);
        expectLine(forkResult.output, /✓ Source\s+neo/);
        expectLine(forkResult.output, /✓ Target\s+neo-v2/);

        // Verify forked agent has core layer on disk
        new MindAssertion(ctx.home, 'neo-v2').exists().hasLayer('core').hasFile('core/profile.md');

        // ── STEP 11: spwn agent export neo ─────────────────────────
        const exportResult = ctx.spwn(['agent', 'export', 'neo']);
        expect(exportResult.exitCode).toBe(0);
        expectLine(exportResult.output, /✓ Exported\s+neo\.tar\.gz/);

        // ── STEP 12: cleanup ───────────────────────────────────────
        const rmForkResult = ctx.spwn(['agent', 'rm', 'neo-v2']);
        expect(rmForkResult.exitCode).toBe(0);
        expectLine(rmForkResult.output, /✓ Deleted agent\s+neo-v2/);

        const rmNeoResult = ctx.spwn(['agent', 'rm', 'neo']);
        expect(rmNeoResult.exitCode).toBe(0);
        expectLine(rmNeoResult.output, /✓ Deleted agent\s+neo/);

        // ── STEP 13: Verify both agents are gone ───────────────────
        const finalLsResult = ctx.spwn(['agent', 'ls']);
        expect(finalLsResult.exitCode).toBe(0);
        const finalOut = stripAnsi(finalLsResult.output);
        const neoRows = finalOut
            .split('\n')
            .filter((l) => /\bneo\b/.test(l) && !l.includes('NAME'));
        expect(neoRows.length).toBe(0);

        expect(existsSync(join(ctx.home, 'agents', 'neo'))).toBe(false);
        expect(existsSync(join(ctx.home, 'agents', 'neo-v2'))).toBe(false);
    });

    test('error recovery: operations on deleted agent fail gracefully', () => {
        // GIVEN - an agent that was created then deleted
        ctx = createTestContext();
        ctx.spwn(['init']);
        ctx.spwn(['agent', 'rm', 'neo']);

        // WHEN/THEN - operations on deleted agent produce clean errors
        const showResult = ctx.spwn(['agent', 'show', 'neo']);
        expect(showResult.exitCode).not.toBe(0);
        expectLine(showResult.output, /agent "neo" not found/);

        const forkResult = ctx.spwn(['agent', 'fork', 'neo', 'neo-copy']);
        expect(forkResult.exitCode).not.toBe(0);

        // Reflect/sleep on missing agent should still handle gracefully (no crash)
        const reflectResult = ctx.spwn(['agent', 'dream', 'neo']);
        expect(reflectResult.output).not.toContain('FATAL');
        expect(reflectResult.output).not.toContain('panic');

        const sleepResult = ctx.spwn(['agent', 'sleep', 'neo']);
        expect(sleepResult.output).not.toContain('FATAL');
        expect(sleepResult.output).not.toContain('panic');
    });

    test('error recovery: down on invalid world ID fails gracefully', () => {
        ctx = createTestContext();
        ctx.spwn(['init']);

        const result = ctx.spwn(['down', 'w-fake-99999']);
        expect(result.exitCode).not.toBe(0);
        expect(result.output).not.toContain('panic');
        expect(result.output).not.toContain('FATAL');
    });

    test('error recovery: double destroy is idempotent', () => {
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['up', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;
        expect(id).toBeTruthy();
        ctx.spwn(['down', id], 30_000);

        const doubleDown = ctx.spwn(['down', id], 30_000);
        expect(doubleDown.output).not.toContain('panic');
        expect(doubleDown.output).not.toContain('FATAL');
    });
});
