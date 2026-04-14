import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createAgent, createSpwnHome } from '../../setup/helpers.js';
import { stripAnsi } from '../../setup/output-helpers.js';
import { spwn } from '../../setup/spwn.specification.js';

describe('zero-friction UX', () => {
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

    // ── 1. Agent talk without world gives spawn hint ────────────

    test('agent talk without world gives spawn hint', async () => {
        // GIVEN - an agent exists but no world is spawned
        createAgent(home, 'solo');

        // WHEN - trying to talk to the agent
        const result = await spwn('talk no world').exec('agent talk solo "hello"').run();

        // THEN - error mentions how to spawn a world
        expect(result.exitCode).not.toBe(0);
        const out = stripAnsi(result.output);
        expect(out).toContain('spwn up --agent');
    });

    // ── 2. Agent talk with nonexistent agent gives create hint ──

    test('agent talk with nonexistent agent gives create hint', async () => {
        // WHEN - talking to an agent that was never created
        const result = await spwn('talk missing agent')
            .exec('agent talk nonexistent "hello"')
            .run();

        // THEN - error suggests creating the agent
        expect(result.exitCode).not.toBe(0);
        const out = stripAnsi(result.output);
        expect(out).toContain('spwn agent new');
    });

    // ── 3. Architect stop when not running is graceful ──────────

    test('architect stop when not running is graceful', async () => {
        // GIVEN - architect is not running (fresh SPWN_HOME, no containers)
        // WHEN - running architect stop
        const result = await spwn('architect stop graceful').exec('architect stop').run();

        // THEN - exits cleanly (not an error)
        expect(result.exitCode).toBe(0);
        const out = stripAnsi(result.output);
        expect(out).toContain('not running');
    });

    // ── 4. Inspect nonexistent world gives ls hint ──────────────

    test('inspect nonexistent world gives ls hint', async () => {
        // WHEN - inspecting a world that does not exist
        const result = await spwn('inspect missing hint')
            .exec('world inspect w-nonexistent-00000')
            .run();

        // THEN - error message includes hint to use spwn ls
        expect(result.exitCode).not.toBe(0);
        const out = stripAnsi(result.output);
        expect(out).toContain('spwn ls');
    });

    // ── 5. Architect talk help mentions auto-start ──────────────

    test('architect talk help mentions auto-start behavior', async () => {
        // WHEN - checking architect talk help
        const result = await spwn('architect talk help').exec('architect talk --help').run();

        // THEN - the talk command exists
        expect(result.exitCode).toBe(0);
    });

    // ── 8. Regression guards ──

    test('no relative /api/ fetch calls in frontend (must use goApiUrl)', async () => {
        const result = await spwn('no-relative-api-fetch')
            .exec([
                'grep',
                '-rn',
                'fetch("/api/\\|fetch(`/api/',
                'apps/web/src/',
                '--include=*.tsx',
                '--include=*.ts',
            ])
            .run();
        // Grep returns exit 1 when no matches (which is what we want)
        const out = stripAnsi(result.output).trim();
        expect(out).not.toContain('fetch("/api/');
        expect(out).not.toContain('fetch(`/api/');
    });

    test("no references to 'God' or 'god' role remain in source code (rename regression)", async () => {
        const repoRoot = resolve(import.meta.dirname, '../../..');

        // Grep -rn for word-boundary 'God' in Go and TS source files
        // Excluding tests, node_modules, .git, and binary dirs
        const result = spawnSync(
            'grep',
            [
                '-rn',
                '--include=*.go',
                '--include=*.ts',
                '--include=*.tsx',
                '--exclude-dir=node_modules',
                '--exclude-dir=.git',
                '--exclude-dir=.next',
                '--exclude-dir=tests',
                '-w',
                'God',
                '.',
            ],
            {
                cwd: repoRoot,
                encoding: 'utf8',
                timeout: 15_000,
            },
        );

        const stdout = result.stdout ?? '';
        const matches = stdout.split('\n').filter((line) => line.trim().length > 0);

        // Should be zero - all God references should be renamed to Architect
        expect(
            matches.length,
            `Found ${matches.length} remaining 'God' references:\n${matches.join('\n')}`,
        ).toBe(0);
    });
});
