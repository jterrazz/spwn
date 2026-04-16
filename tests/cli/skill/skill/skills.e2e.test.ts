import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * System skills / AGENTS.md injection — docker-backed.
 *
 * Merged from the legacy `skills-docker.e2e.test.ts`. Every spawned
 * world gets a canonical set of system files bind-mounted or written
 * into `/world/`:
 *
 *   - `/world/AGENTS.md`   — the operating manual content
 *   - `/world/roster.md`   — regenerated per spawn, names the agents
 *   - `/world/skills/`     — shared system skill bundle
 *
 * Lives under the cli vitest project (`cli/system/**`) but spec
 * Still spawns real containers; cleanup runs via the test-run label.
 */
describe('system skills infrastructure (docker)', () => {
    test('AGENTS.md, roster.md, and /world/skills are laid down on up', async () => {
        await using result = await spec('skills layout').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Created container');

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // AGENTS.md exists and has meaningful content.
        const agents = neo.file('/world/AGENTS.md');
        expect(agents.exists).toBe(true);
        expect(agents.content.length).toBeGreaterThan(100);

        // Roster.md exists and references the neo agent.
        const roster = neo.file('/world/roster.md');
        expect(roster.exists).toBe(true);
        expect(roster.content).toContain('neo');

        // Skills directory exists and is non-empty.
        expect(neo.file('/world/skills').exists).toBe(true);
        const ls = await neo.exec('ls /world/skills');
        expect(ls.exitCode).toBe(0);
        expect(ls.stdout.text.trim().length).toBeGreaterThan(0);
    });

    test('agent can read /world/skills from inside the container', async () => {
        await using result = await spec('skills readable').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');

        // `test -d` succeeds on a directory — this is what the legacy
        // `toHaveDirectory('/world/skills')` helper reduced to.
        const testDir = await neo.exec('test -d /world/skills');
        expect(testDir.exitCode).toBe(0);

        // And the agent user (uid `spwn`) can list it.
        const ls = await neo.exec('ls -1 /world/skills');
        expect(ls.exitCode).toBe(0);
    });
});

describe('spwn skill new (project-local)', () => {
    test('skill new inside a project writes into the project tree', async () => {
        // Given - an initialised empty project
        // When - we author a new skill
        // Then - it lands under spwn/skills/ (not ~/.spwn/skills/)
        const result = await spec('project-scoped skill new')
            .project('empty')
            .exec(['init', 'skill new my-skill'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/skills/my-skill.md').exists).toBe(true);
    });
});
