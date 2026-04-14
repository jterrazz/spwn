import { afterEach, describe, expect, test } from 'vitest';

import { expectLine } from '../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../setup/spwn.specification.js';

describe('graceful shutdown', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test('destroy writes journal entry for the agent', () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;
        expect(id).toBeTruthy();
        ctx.world(id).toBeRunning();

        // WHEN - destroying it
        const destroyResult = ctx.spwn(['down', id], 30_000);
        expect(destroyResult.exitCode).toBe(0);

        // THEN - journal entry exists with world ID
        ctx.mind('neo').hasJournalEntries(1);
        ctx.mind('neo').journalContains(id);
    });

    test('down --all destroys all running worlds', () => {
        // GIVEN - two spawned worlds (same agent for simplicity)
        ctx = createTestContext();
        ctx.spwn(['init']);

        const spawn1 = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id1 = parseWorldId(spawn1.output)!;
        expect(id1).toBeTruthy();

        const spawn2 = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id2 = parseWorldId(spawn2.output)!;
        expect(id2).toBeTruthy();

        // WHEN - destroying all worlds
        const downResult = ctx.spwn(['down', '--all'], 30_000);

        // THEN - exits successfully
        expect(downResult.exitCode).toBe(0);
        expectLine(downResult.output, /world\(s\) destroyed/);

        // AND - both worlds are gone
        ctx.world(id1).toNotExist();
        ctx.world(id2).toNotExist();
        ctx.state().noWorld(id1);
        ctx.state().noWorld(id2);

        // AND - journal entries exist for the agent
        ctx.mind('neo').hasJournalEntries(2);
    });

    test('destroy updates agent status - world removed from list', () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;
        expect(id).toBeTruthy();

        // Verify it exists in state
        ctx.state().hasWorld(id);

        // WHEN - destroying
        ctx.spwn(['down', id], 30_000);

        // THEN - world is gone from state and list
        ctx.state().noWorld(id);
        const listResult = ctx.spwn(['ls']);
        expect(listResult.exitCode).toBe(0);
        // The destroyed world should not appear in ls
        if (listResult.output.includes(id)) {
            throw new Error(`Expected world ${id} to be gone from list`);
        }
    });

    test('down --all with no running worlds succeeds gracefully', () => {
        // GIVEN - an initialized SPWN_HOME with no worlds
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - running down --all
        const result = ctx.spwn(['down', '--all'], 30_000);

        // THEN - exits successfully with 0 worlds destroyed
        expect(result.exitCode).toBe(0);
        expectLine(result.output, /0 world\(s\) destroyed/);
    });

    test('upgrade command exists and shows help', () => {
        // GIVEN - an initialized context
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - running upgrade --help
        const result = ctx.spwn(['upgrade', '--help']);

        // THEN - shows upgrade help text
        expect(result.exitCode).toBe(0);
        expectLine(result.output, /[Dd]ownloads.*spwn release/);
        expectLine(result.output, /[Rr]unning worlds are stopped/);
    });
});
