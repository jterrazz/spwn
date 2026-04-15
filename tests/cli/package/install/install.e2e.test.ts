import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Coverage for `spwn package install` / `spwn package uninstall` —
 * the npm-style dependency-management verbs. These mutate the target
 * agent.yaml plus the project-root spwn.lock.yaml and never touch
 * Docker, so the tests run fast against the lightweight docker-pilot
 * fixture.
 *
 * What's locked in here:
 *   - installing an @spwn/* ref pins it in spwn.lock.yaml
 *   - installing a bare name is rejected with an authoring hint
 *   - installing @<owner>/* is rejected as unsupported
 *   - uninstall removes the ref from agent.yaml and the lockfile
 *   - double-install is idempotent (no duplicate agent.yaml entry)
 */
describe('spwn package install', () => {
    test('pins an @spwn/* ref into spwn.lock.yaml', async () => {
        const result = await spec('package install builtin')
            .project('docker-pilot')
            .exec('package install @spwn/python')
            .run();

        expect(result.exitCode).toBe(0);
        const lock = result.file('spwn.lock.yaml');
        expect(lock.exists).toBe(true);
        expect(lock.content).toContain('@spwn/python');
        expect(lock.content).toContain('source: builtin');

        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).toContain('@spwn/python');
    });

    test('rejects a bare name with an authoring hint', async () => {
        const result = await spec('package install bare rejected')
            .project('docker-pilot')
            .exec('package install my-local-tool')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('bare name');
        expect(result.stderr.text).toContain('spwn/packages/');
    });

    test('rejects @<owner>/* as unsupported', async () => {
        const result = await spec('package install registry rejected')
            .project('docker-pilot')
            .exec('package install @acme/foo')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('not yet supported');
    });

    test('rejects an unknown @spwn/* ref', async () => {
        const result = await spec('package install unknown builtin')
            .project('docker-pilot')
            .exec('package install @spwn/nonesuch')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('unknown builtin');
    });

    test('is idempotent on re-install', async () => {
        const result = await spec('package install idempotent')
            .project('docker-pilot')
            .exec(['package install @spwn/python', 'package install @spwn/python'])
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        const count = (agentYaml.content.match(/@spwn\/python/g) ?? []).length;
        expect(count).toBe(1);
    });

    test('pkg alias works identically to the full verb', async () => {
        const result = await spec('pkg alias')
            .project('docker-pilot')
            .exec('pkg install @spwn/python')
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.lock.yaml').content).toContain('@spwn/python');
    });
});

describe('spwn package uninstall', () => {
    test('removes the ref from agent.yaml and the lockfile', async () => {
        const result = await spec('package uninstall')
            .project('docker-pilot')
            .exec(['package install @spwn/python', 'package uninstall @spwn/python'])
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).not.toContain('@spwn/python');

        const lock = result.file('spwn.lock.yaml');
        if (lock.exists) {
            expect(lock.content).not.toContain('@spwn/python');
        }
    });
});

describe('spwn package ls', () => {
    test('shows an empty state on a project with no installs', async () => {
        const result = await spec('package ls empty')
            .project('docker-pilot')
            .exec('package ls')
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('No packages installed');
    });

    test('lists installed packages after install', async () => {
        const result = await spec('package ls after install')
            .project('docker-pilot')
            .exec(['package install @spwn/python', 'package ls'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('@spwn/python');
    });
});
