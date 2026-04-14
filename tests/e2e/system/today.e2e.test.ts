import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
/**
 * E2E tests for CLI-only behavior (no Docker required).
 *
 * Covers: agent identity structure, org.yaml removal, CLI upgrade --check.
 *
 * NOTE: example install + gallery tests were removed when `spwn example`
 * was deleted. Template install now flows through /api/templates and will
 * get a CLI entrypoint via `spwn init @spwn/<slug>` in a followup.
 */
import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';
import { stripAnsi } from '../../setup/output-helpers.js';
import { spwn } from '../../setup/spwn.specification.js';

describe('org.yaml removal', () => {
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

    test('init does NOT create org.yaml', async () => {
        // WHEN - running init
        const result = await spwn('init no org').exec('init').run();

        // THEN - no org.yaml created
        expect(result.exitCode).toBe(0);
        expect(existsSync(join(home, 'org.yaml'))).toBe(false);
        expect(result.output).not.toContain('org.yaml');
    });
});

describe('agent mind structure', () => {
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

    test('agent new creates the 5-layer Mind structure', async () => {
        // WHEN - creating a new agent
        const result = await spwn('agent new').exec('agent new TestAgent').run();

        // THEN - the 5 Mind layers are created
        expect(result.exitCode).toBe(0);
        const agentDir = join(home, 'agents', 'TestAgent');
        for (const layer of ['core', 'skills', 'knowledge', 'playbooks', 'journal']) {
            expect(existsSync(join(agentDir, layer)), `missing ${layer}/`).toBe(true);
        }

        // Core/profile.md exists with content
        const personaPath = join(agentDir, 'core', 'profile.md');
        expect(existsSync(personaPath)).toBe(true);
        const persona = readFileSync(personaPath, 'utf8');
        expect(persona.length).toBeGreaterThan(10);

        // Old structure should NOT exist
        expect(existsSync(join(agentDir, 'identity'))).toBe(false);
        expect(existsSync(join(agentDir, 'memory'))).toBe(false);
        expect(existsSync(join(agentDir, 'sessions'))).toBe(false);
    });
});

describe('CLI upgrade', () => {
    test('upgrade --check finds latest version', async () => {
        // WHEN - checking for updates
        const result = await spwn('upgrade check').exec('upgrade --check').run();

        // THEN - reports a version (the local build is "dev" so latest != current)
        const output = stripAnsi(result.output);
        expect(output).toMatch(/Latest version\s+v\d+\.\d+\.\d+/);
    });
});
