import { afterEach, describe, expect, test } from 'vitest';

import {
    expectLine,
    expectNoLine,
    expectTableHeader,
    stripAnsi,
} from '../../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../../setup/spwn.specification.js';

/**
 * Docker-backed CLI execution tests.
 *
 * These exercise the world lifecycle (up/down/ls/logs/inspect/snap),
 * which requires a running Docker daemon. They still use the legacy
 * `createTestContext` / `ctx.spwn(...)` helpers because the current
 * `@jterrazz/test` ExecAdapter cannot drive long-running container
 * orchestration cleanly. Migrating them is deferred until the
 * framework grows a proper Docker adapter — not in scope for this
 * spec-runner batch.
 */
describe('CLI execution - world aliases (Docker)', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test("'spwn up' alias spawns a world", () => {
        // GIVEN - initialized context
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - using the 'up' alias
        const result = ctx.spwn(['up', '--agent', 'neo', '-w', ctx.home], 60_000);

        // THEN - world is created
        expect(result.exitCode).toBe(0);
        const id = parseWorldId(result.output)!;
        expect(id).toBeTruthy();

        // AND - appears in ls
        const listResult = ctx.spwn(['ls']);
        expect(listResult.exitCode).toBe(0);
        expect(stripAnsi(listResult.output)).toContain(id);
    });

    test("'spwn down' alias destroys a world", () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['up', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;
        expect(id).toBeTruthy();

        // WHEN - using the 'down' alias
        const destroyResult = ctx.spwn(['down', id], 30_000);

        // THEN - world is destroyed
        expect(destroyResult.exitCode).toBe(0);
        expectLine(destroyResult.output, /✓ World destroyed\. Agent survives\./);

        // AND - world gone from ls
        const listResult = ctx.spwn(['ls']);
        expect(listResult.exitCode).toBe(0);
        expectNoLine(
            listResult.output,
            new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)),
        );
    });

    test("'spwn ls' alias lists worlds", () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;

        // WHEN - using the 'ls' alias
        const listResult = ctx.spwn(['ls']);

        // THEN - world appears in output
        expect(listResult.exitCode).toBe(0);
        expect(stripAnsi(listResult.output)).toContain(id);
        expectTableHeader(listResult.output, ['ID', 'CONFIG', 'AGENTS', 'STATUS']);
    });

    test("'spwn logs' alias works for world", () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;

        // WHEN - using the 'logs' command (world logs)
        const logsResult = ctx.spwn(['world', 'logs', id]);

        // THEN - doesn't error (agent may not have output yet)
        expect(logsResult.exitCode).toBe(0);
        // AND - output is a string (may be empty if agent hasn't logged yet)
        expect(typeof logsResult.output).toBe('string');
    });

    test("'spwn inspect' works for world via world inspect", () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;

        // WHEN - inspecting the world
        const inspectResult = ctx.spwn(['world', 'inspect', id]);

        // THEN - output contains world details
        expect(inspectResult.exitCode).toBe(0);
        const out = stripAnsi(inspectResult.output);
        expect(out).toContain(id);
        expect(out).toContain('default'); // Config name
        expect(out).toContain('neo'); // Agent
        expectLine(inspectResult.output, /Config:\s+default/);
        expectLine(inspectResult.output, /Status:\s+(running|idle)/);
    });

    test("'spwn snap save' creates snapshot", () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;

        // WHEN - saving via spwn snap save
        const snapResult = ctx.spwn(['snap', 'save', id]);

        // THEN - snapshot created
        expect(snapResult.exitCode).toBe(0);
        expectLine(snapResult.output, /[Ss]aved snapshot|[Ss]nap(shot)? saved/);
    });
});
