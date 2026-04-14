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
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).toContain('install');
        expect(combined).toContain('ls');
        expect(combined).toContain('search');
        expect(combined).toContain('rm');
    });

    test("'spwn get ls' shows empty list when no packages installed", async () => {
        const result = await isolated('get ls').exec('get ls').run();

        expect(result.exitCode).toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).toContain('No packages installed');
    });

    test("'spwn get install nonexistent' handles gracefully", async () => {
        const result = await isolated('get install nonexistent')
            .exec('get install nonexistent')
            .run();

        // Placeholder surface — command may succeed with a stub or
        // Fail cleanly. The requirement is "no crash / no stack trace".
        const combined = result.stdout.text + result.stderr.text;
        expect(combined.length).toBeGreaterThan(0);
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('ReferenceError');
        expect(combined).not.toContain('panic:');
    });

    test("'spwn get rm nonexistent' handles gracefully", async () => {
        const result = await isolated('get rm nonexistent').exec('get rm nonexistent').run();

        const combined = result.stdout.text + result.stderr.text;
        expect(combined.length).toBeGreaterThan(0);
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('ReferenceError');
        expect(combined).not.toContain('panic:');
    });

    test("'spwn get search' handles search query", async () => {
        const result = await isolated('get search test').exec('get search test').run();

        const combined = result.stdout.text + result.stderr.text;
        expect(combined.length).toBeGreaterThan(0);
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('ReferenceError');
        expect(combined).not.toContain('panic:');
    });

    test("'spwn get' without subcommand shows help-ish output", async () => {
        const result = await isolated('get bare').exec('get').run();

        const combined = result.stdout.text + result.stderr.text;
        expect(combined.length).toBeGreaterThan(0);
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('panic:');
        expect(combined).toMatch(/install|search|ls|rm|help/i);
    });
});
