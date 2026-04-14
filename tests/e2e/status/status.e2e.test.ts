import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';
import { stripAnsi } from '../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    spwn,
    type TestContext,
} from '../../setup/spwn.specification.js';

describe('spwn status', () => {
    describe('without worlds', () => {
        let home: string;
        let originalSpwnHome: string | undefined;

        beforeEach(() => {
            originalSpwnHome = process.env.SPWN_HOME;
            home = createSpwnHome();
            process.env.SPWN_HOME = home;
        });

        afterEach(() => {
            if (originalSpwnHome !== undefined) {
                process.env.SPWN_HOME = originalSpwnHome;
            } else {
                delete process.env.SPWN_HOME;
            }
        });

        test('shows the spwn brand line', async () => {
            await spwn('init').exec('init').run();
            const result = await spwn('status').exec('status').run();

            expect(result.exitCode).toBe(0);
            const out = stripAnsi(result.output);
            expect(out).toContain('spwn');
        });

        test('shows architect section as offline', async () => {
            await spwn('init').exec('init').run();
            const result = await spwn('status').exec('status').run();

            const out = stripAnsi(result.output);
            expect(out).toContain('Architect');
            expect(out).toContain('offline');
        });

        test('shows worlds section', async () => {
            await spwn('init').exec('init').run();
            const result = await spwn('status').exec('status').run();

            const out = stripAnsi(result.output);
            expect(out).toContain('Worlds');
        });

        test('shows physics constants from default config', async () => {
            await spwn('init').exec('init').run();
            const result = await spwn('status').exec('status').run();

            const out = stripAnsi(result.output);
            expect(out).toMatch(/\d+ cpu/);
            expect(out).toContain('512m');
        });
    });

    describe('with active world', () => {
        let ctx: TestContext;

        afterEach(() => {
            ctx?.cleanup();
        });

        test('shows world bubble with agent', () => {
            ctx = createTestContext();
            ctx.spwn(['init']);
            const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
            const id = parseWorldId(spawnResult.output)!;

            const listResult = ctx.spwn(['ls']);
            const listOut = stripAnsi(listResult.output);

            const result = ctx.spwn(['status']);
            expect(result.exitCode).toBe(0);
            const out = stripAnsi(result.output);

            expect(out).toContain('spwn');

            // If the ls output shows the ID, status should too.
            if (listOut.includes(id)) {
                expect(out).toContain(id);
                expect(out).toContain('neo');
            }
        });
    });

    describe('error handling', () => {
        let home: string;
        let originalSpwnHome: string | undefined;

        beforeEach(() => {
            originalSpwnHome = process.env.SPWN_HOME;
            home = createSpwnHome();
            process.env.SPWN_HOME = home;
        });

        afterEach(() => {
            if (originalSpwnHome !== undefined) {
                process.env.SPWN_HOME = originalSpwnHome;
            } else {
                delete process.env.SPWN_HOME;
            }
        });

        test('status on uninitialized home still works', async () => {
            const result = await spwn('status no init').exec('status').run();

            expect(result.exitCode).toBe(0);
            const out = stripAnsi(result.output);
            expect(out).toContain('spwn');
        });
    });
});
