import { execSync } from 'node:child_process';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn agent export` / `spwn agent import`.
 *
 * Export writes `<name>.tar.gz` into the current working directory,
 * which for each spec is a fresh temp project — so the archive lands
 * next to `spwn-home/` and we can poke it with `tar tzf` via
 * `result.file(...).path`.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn agent export', () => {
    test('export creates a tar.gz next to the project', async () => {
        const result = await isolated('export neo')
            .exec(['agent create neo', 'agent export neo'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('neo.tar.gz').exists).toBe(true);
        expect(result.stderr.text + result.stdout.text).toContain('Exported');
    });

    test('exported archive contains all Mind layers', async () => {
        const result = await isolated('export contents')
            .exec(['agent create neo', 'agent export neo'])
            .run();

        expect(result.exitCode).toBe(0);
        const tarPath = join(result.filesystem.cwd, 'neo.tar.gz');
        const listing = execSync(`tar tzf ${tarPath}`, { encoding: 'utf8' });
        expect(listing).toContain('core/profile.md');
        expect(listing).toMatch(/(^|\n)core(\/|\n|$)/);
        expect(listing).toMatch(/(^|\n)skills(\/|\n|$)/);
        expect(listing).toMatch(/(^|\n)knowledge(\/|\n|$)/);
        expect(listing).toMatch(/(^|\n)playbooks(\/|\n|$)/);
        expect(listing).toMatch(/(^|\n)journal(\/|\n|$)/);
    });

    test('export --exclude still succeeds', async () => {
        const result = await isolated('export with exclude')
            .exec(['agent create neo', 'agent export neo --exclude journal,sessions'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('neo.tar.gz').exists).toBe(true);
        expect(result.stderr.text + result.stdout.text).toContain('Exported');
    });

    test('export on a missing agent errors cleanly', async () => {
        const result = await isolated('export missing').exec('agent export ghost').run();

        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).toContain('agent "ghost" not found');
        expect(combined).not.toContain('panic:');
    });

    test('import restores an agent from its own export', async () => {
        // Round-trip: create -> export -> rm -> import. Archive name
        // Drives the restored agent name, so we reuse "neo" here
        // (the CLI dropped the old `--name` override).
        const result = await isolated('import round trip')
            .exec([
                'agent create neo',
                'agent export neo',
                'agent rm neo',
                'agent import neo.tar.gz',
            ])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/neo/core/profile.md').exists).toBe(true);
        expect(result.stderr.text + result.stdout.text).toContain('Imported agent');
    });
});
