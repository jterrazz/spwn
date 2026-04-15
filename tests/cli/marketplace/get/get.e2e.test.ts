import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Marketplace `spwn get *` subcommand smoke tests.
 *
 * `spwn get` is currently a planned / placeholder surface — most
 * subcommands exit zero with a stub message. These tests pin down
 * the command group's shape and guarantee we never regress to
 * a crash/stack-trace.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('marketplace - spwn get', () => {
    test("'spwn get --help' shows subcommands", async () => {
        const result = await isolated('get help').exec('get --help').run();

        expect(result.exitCode).toBe(0);
        // Cobra --help renders on stdout.
        result.stdout.toContain('install');
        result.stdout.toContain('ls');
        result.stdout.toContain('search');
        result.stdout.toContain('rm');
    });

    test("'spwn get ls' shows empty list when no packages installed", async () => {
        const result = await isolated('get ls').exec('get ls').run();

        expect(result.exitCode).toBe(0);
        // Stub placeholder — the "No packages installed" line lands on stderr
        // (ui.Info). Fall back to both streams to stay resilient as the
        // Command moves from placeholder to real implementation.
        expect(result.stdout.text + result.stderr.text).toContain('No packages installed');
    });

    test("'spwn get install nonexistent' exits non-zero with a not-implemented banner", async () => {
        // Given - isolated home, bogus package name
        // When - running the unimplemented get install stub
        // Then - exit 2 (feature-unavailable), no crash, banner on stderr
        const result = await isolated('get install nonexistent')
            .exec('get install nonexistent')
            .run();

        expect(result.exitCode).toBe(2);
        expect(result.stderr.text).toMatch(/not yet implemented/i);
        expect(result.stderr.text).not.toContain('panic:');
    });

    test("'spwn get rm nonexistent' handles gracefully", async () => {
        const result = await isolated('get rm nonexistent').exec('get rm nonexistent').run();

        expect(result.stdout.text.length + result.stderr.text.length).toBeGreaterThan(0);
        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.stderr.text).not.toContain('ReferenceError');
        expect(result.stderr.text).not.toContain('panic:');
    });

    test("'spwn get search' handles search query", async () => {
        const result = await isolated('get search test').exec('get search test').run();

        expect(result.stdout.text.length + result.stderr.text.length).toBeGreaterThan(0);
        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.stderr.text).not.toContain('ReferenceError');
        expect(result.stderr.text).not.toContain('panic:');
    });

    test("'spwn tool show' exits non-zero with a not-implemented banner", async () => {
        // Given - tool show is planned but not yet wired
        // When - invoking it on any pack name
        // Then - exit 2, not-implemented banner on stderr
        const result = await isolated('tool show unimplemented').exec('tool show @spwn/unix').run();

        expect(result.exitCode).toBe(2);
        expect(result.stderr.text).toMatch(/not yet implemented/i);
    });

    test("'spwn get' without subcommand shows help-ish output", async () => {
        const result = await isolated('get bare').exec('get').run();

        // Bare command group prints help-ish output; exact stream depends on
        // Whether cobra (stdout) or our stub renderer (stderr) handled it.
        expect(result.stdout.text.length + result.stderr.text.length).toBeGreaterThan(0);
        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.stderr.text).not.toContain('panic:');
        expect(result.stdout.text + result.stderr.text).toMatch(/install|search|ls|rm|help/i);
    });
});
