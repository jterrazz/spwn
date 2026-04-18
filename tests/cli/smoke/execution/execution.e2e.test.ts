import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * CLI execution — agent management, flags, enter, status and the
 * `world aliases` docker block at the bottom.
 *
 * spwn's success path writes status banners to stderr (Unix
 * convention). Stable banners get stderr snapshots under
 * `./expected/stderr/`; machine-dependent output (paths, ids) is
 * matched with substrings against `result.stderr.text`.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

// ── Agent management (no Docker) ─────────────────────────────

describe('CLI execution - agent commands', () => {
    test("'spwn agent create testbot' prints the creation banner", async () => {
        const result = await isolated('agent create testbot').exec('agent create testbot').run();
        expect(result.exitCode).toBe(0);
        await result.stderr.toMatch('agent-create-testbot.txt');
    });

    test("'spwn agent rm' on missing agent errors cleanly", async () => {
        /*
         * Each spec gets a fresh temp workdir, so agent create/rm pairs
         * across specs cannot share state. The legacy "create-then-rm"
         * round-trip test is covered implicitly by the Go unit tests;
         * here we just verify rm fails cleanly without a stack trace.
         */
        const result = await isolated('agent rm missing').exec('agent rm ghost').run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toContain('panic:');
        expect(result.stderr.text).not.toContain('goroutine ');
    });

    test("'spwn agent ls' on an empty home prints the empty state", async () => {
        const result = await isolated('agent ls empty').exec('agent ls').run();
        expect(result.exitCode).toBe(0);
        await result.stderr.toMatch('agent-ls-empty.txt');
    });

    test("'spwn agent show' on nonexistent agent errors cleanly", async () => {
        const result = await isolated('agent show missing').exec('agent show ghost').run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toContain('panic:');
        expect(result.stderr.text).not.toContain('goroutine ');
    });
});

// ── Enter command ──────────────────────────────────────────

describe('CLI execution - enter command', () => {
    test("'spwn world enter <nonexistent-id>' returns clean error", async () => {
        const result = await isolated('enter nonexistent').exec('world enter world-fake-99999').run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toContain('panic:');
        expect(result.stderr.text).not.toContain('goroutine ');
    });

    test("'spwn world enter --help' shows usage", async () => {
        // --help is one of the few spwn subcommands that writes to stdout
        // (cobra's default behaviour), so we can snapshot it.
        const result = await isolated('enter help').exec('world enter --help').run();

        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('enter');
        expect(out).toContain('world-id');
    });
});

// ── Global flags ─────────────────────────────────────────────

describe('CLI execution - global flags', () => {
    test("'--version' shows version string", async () => {
        // --version writes to stdout.
        const result = await isolated('version').exec('--version').run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/spwn version/);
    });
});

// ── Status command ──────────────────────────────────────────

describe('CLI execution - status command', () => {
    test("'spwn status' runs cleanly after init", async () => {
        /*
         * Richer status-output coverage lives in
         * tests/cli/status/status/*. Here we just confirm both
         * commands emit their stable banners.
         */
        const initResult = await isolated('init for status').exec('init').run();
        expect(initResult.exitCode).toBe(0);
        // `init` banner includes the basename of the temp workdir,
        // Which is not snapshot-stable (the transform only masks the
        // Path prefix, not the basename). Assert on the marker.
        expect(initResult.stderr.text).toContain('Initialised spwn project');

        const statusResult = await isolated('status after init').exec('status').run();
        expect(statusResult.exitCode).toBe(0);
        const stderr = statusResult.stderr.text;
        expect(stderr).toContain('Worlds');
        expect(stderr).toContain('Architect');
    });

    test("'spwn auth' from an empty project still renders the provider table", async () => {
        const result = await isolated('auth from execution').exec('auth').run();
        expect(result.exitCode).toBe(0);
        // Provider table is keychain-dependent (see auth.e2e.test.ts),
        // So we only assert on the stable header row here.
        expect(result.stderr.text).toContain('PROVIDER');
    });
});

// ── Docker-backed world lifecycle (aliases) ─────────────────
//
// Merged in from the legacy `execution-docker.e2e.test.ts`. These
// Exercise `up`, `down`, `ls`, `world inspect`, `world logs`, and
// `snap save` against a real container. The file lives in the cli
// Vitest project, but spec runs real docker regardless — the
// Cleanup label + `await using` still apply.
describe('CLI execution - world aliases (docker)', () => {
    test("'spwn up' spawns a world that appears in world list --json", async () => {
        await using result = await spec('up alias')
            .project('docker-pilot')
            .exec(['up', 'world list --json'])
            .run();

        expect(result.exitCode).toBe(0);

        const list = result.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
        expect(list.worlds).toHaveLength(1);
        expect(list.worlds[0].name).toBe('neo');
        expect(list.worlds[0].status).toBe('running');

        // And the container really exists.
        expect(result.container('neo').running).toBe(true);
    });

    test("'spwn down' destroys a spawned world", async () => {
        await using result = await spec('down alias')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Destroyed');
        result.stderr.toContain('project world(s) destroyed');
        expect(result.container('neo').exists).toBe(false);
    });

    test("'spwn world inspect' surfaces status for a running world", async () => {
        // Step 1: up, capture the spwn world id from the container label.
        await using up = await spec('inspect up').project('docker-pilot').exec('up').run();

        expect(up.exitCode).toBe(0);
        const neo = up.container('neo');
        const worldId = (neo.inspect.value as { Config?: { Labels?: Record<string, string> } })
            .Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();

        // Step 2: world inspect <id>
        await using inspect = await spec('inspect call')
            .project('docker-pilot')
            .exec(`world inspect ${worldId}`)
            .run();

        expect(inspect.exitCode).toBe(0);
        // `world inspect` renders via stepper (stderr).
        inspect.stderr.toContain(worldId!);
        expect(inspect.stderr.text).toMatch(/Status/);
    });

    test("'spwn world logs' returns cleanly for a running world", async () => {
        await using up = await spec('logs up').project('docker-pilot').exec('up').run();
        expect(up.exitCode).toBe(0);
        const worldId = (
            up.container('neo').inspect.value as {
                Config?: { Labels?: Record<string, string> };
            }
        ).Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();

        await using logs = await spec('logs call')
            .project('docker-pilot')
            .exec(`world logs ${worldId}`)
            .run();

        // Agent may not have emitted anything yet — we just require the
        // Command to exit cleanly.
        expect(logs.exitCode).toBe(0);
    });

    test("'spwn snap save' creates a snapshot of a running world", async () => {
        await using up = await spec('snap up').project('docker-pilot').exec('up').run();
        expect(up.exitCode).toBe(0);
        const worldId = (
            up.container('neo').inspect.value as {
                Config?: { Labels?: Record<string, string> };
            }
        ).Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();

        await using snap = await spec('snap save')
            .project('docker-pilot')
            .exec(`world snap save ${worldId}`)
            .run();

        expect(snap.exitCode).toBe(0);
        // Snapshot save banner lands on stderr.
        expect(snap.stderr.text).toMatch(/[Ss]aved|[Ss]nap(shot)? saved|created/);
    });
});
