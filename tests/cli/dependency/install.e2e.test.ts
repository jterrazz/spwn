import { describe, expect, test } from 'vitest';

import { spec } from '../../setup/cli.specification.js';

/**
 * Coverage for `spwn install` / `spwn uninstall` —
 * the npm-style dependency-management verbs. These mutate the target
 * agent.yaml plus the project-root spwn.lock and never touch
 * Docker, so the tests run fast against the lightweight docker-pilot
 * fixture.
 *
 * What's locked in here:
 *   - installing an spwn:* ref pins it in spwn.lock
 *   - installing a bare name is rejected with an authoring hint
 *   - installing @<owner>/* is rejected as unsupported
 *   - uninstall removes the ref from agent.yaml and the lockfile
 *   - double-install is idempotent (no duplicate agent.yaml entry)
 */
describe('spwn install', () => {
    test('pins an spwn:* ref into spwn.lock', async () => {
        const result = await spec('install builtin')
            .project('docker-pilot')
            .exec('install spwn:python')
            .run();

        expect(result.exitCode).toBe(0);
        const lock = result.file('spwn.lock');
        expect(lock.exists).toBe(true);
        expect(lock.content).toContain('spwn:python');

        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).toContain('spwn:python');
    });

    test('rejects a bare name with an authoring hint', async () => {
        const result = await spec('install bare rejected')
            .project('docker-pilot')
            .exec('install my-local-tool')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('bare name');
        expect(result.stderr.text).toContain('spwn/tools/');
    });

    test('rejects @<owner>/* as unsupported', async () => {
        const result = await spec('install registry rejected')
            .project('docker-pilot')
            .exec('install @acme/foo')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('not yet supported');
    });

    test('rejects an unknown spwn:* ref', async () => {
        const result = await spec('install unknown builtin')
            .project('docker-pilot')
            .exec('install spwn:nonesuch')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('unknown builtin');
    });

    test('is idempotent on re-install', async () => {
        const result = await spec('install idempotent')
            .project('docker-pilot')
            .exec(['install spwn:python', 'install spwn:python'])
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        const count = (agentYaml.content.match(/spwn:python/g) ?? []).length;
        expect(count).toBe(1);
    });
});

describe('spwn uninstall', () => {
    test('removes the ref from agent.yaml and the lockfile', async () => {
        const result = await spec('uninstall')
            .project('docker-pilot')
            .exec(['install spwn:python', 'uninstall spwn:python'])
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).not.toContain('spwn:python');

        const lock = result.file('spwn.lock');
        if (lock.exists) {
            expect(lock.content).not.toContain('spwn:python');
        }
    });
});
