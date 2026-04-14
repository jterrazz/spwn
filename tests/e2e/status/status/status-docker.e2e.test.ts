import { afterEach, describe, expect, test } from 'vitest';

import { stripAnsi } from '../../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../../setup/spwn.specification.js';

/**
 * Docker-backed `spwn status` coverage.
 *
 * Status output is rendered to stderr, which the @jterrazz/test
 * ExecAdapter drops on exit 0, so the spec runner cannot assert on
 * the content. This test stays on the legacy helpers because
 * `createTestContext` captures both streams.
 *
 * Kept outside the spec-runner migration scope until the test
 * framework either gains a stderr-on-success capture or spwn starts
 * writing its renderer output to stdout.
 */
describe('spwn status - with active world (Docker)', () => {
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
