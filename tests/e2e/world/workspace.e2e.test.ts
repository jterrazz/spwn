import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, describe, expect, test } from 'vitest';

import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../setup/spwn.specification.js';

describe('world workspace persistence', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test('host files are visible inside container', () => {
        // GIVEN - a workspace with a test file
        ctx = createTestContext();
        ctx.spwn(['init']);
        writeFileSync(join(ctx.home, 'host-file.txt'), 'created on host');

        // WHEN - spawning a world with that workspace
        const result = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        expect(result.exitCode).toBe(0);
        const id = parseWorldId(result.output)!;

        // THEN - the file exists inside the container at /work/default/
        ctx.world(id).toHaveFile('/work/default/host-file.txt', 'created on host');
    });

    test('container changes persist to host', () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);

        const result = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        expect(result.exitCode).toBe(0);
        const id = parseWorldId(result.output)!;

        // WHEN - creating a file inside the container
        ctx.world(id).exec("echo 'created in container' > /work/default/container-file.txt");

        // THEN - the file exists on the host filesystem
        const hostPath = join(ctx.home, 'container-file.txt');
        expect(existsSync(hostPath)).toBe(true);
        const content = readFileSync(hostPath, 'utf8').trim();
        expect(content).toBe('created in container');
    });

    test('workspace mount is read-write', () => {
        // GIVEN - a spawned world
        ctx = createTestContext();
        ctx.spwn(['init']);

        const result = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        expect(result.exitCode).toBe(0);
        const id = parseWorldId(result.output)!;

        // WHEN - writing and reading back inside container
        ctx.world(id).exec("echo 'rw-test-content' > /work/default/rw-test.txt");
        const readBack = ctx.world(id).exec('cat /work/default/rw-test.txt');

        // THEN - content matches
        expect(readBack).toBe('rw-test-content');
    });

    test('spawn with non-existent workspace path fails gracefully', () => {
        // GIVEN - an initialized context
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - spawning with a workspace path that doesn't exist
        const result = ctx.spwn(
            ['world', '--agent', 'neo', '-w', '/tmp/nonexistent-path-12345'],
            60_000,
        );

        // THEN - exits with error (no stack trace)
        expect(result.exitCode).not.toBe(0);
        expect(result.output).not.toContain('TypeError');
        expect(result.output).not.toContain('FATAL');
    });
});
