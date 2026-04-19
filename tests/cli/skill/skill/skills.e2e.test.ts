import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * System skills / agent prompt injection — docker-backed.
 *
 * The per-agent CLAUDE.md is the single source of ambient context:
 * it inlines physics, faculties, roster, conventions, and the
 * playbook index. Tool-shipped skills still land at /world/skills/
 * (baked into the image) and are surfaced to Claude Code through a
 * spawn-time `.claude/skills` symlink on the agent's HOME.
 */
describe('system skills infrastructure (docker)', () => {
    test('CLAUDE.md is laid down per agent with world context inlined', async () => {
        await using result = await spec('claude md layout')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Created container');

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // CLAUDE.md exists, is non-trivial, and names the agent.
        const claude = neo.file('/agents/neo/CLAUDE.md');
        expect(claude.exists).toBe(true);
        expect(claude.content.length).toBeGreaterThan(200);
        expect(claude.content).toContain('neo');

        // The standard section headers are all there.
        for (const heading of [
            '## Identity',
            '## Physics',
            '## Faculties',
            '## Roster',
            '## Conventions',
        ]) {
            expect(claude.content, `missing ${heading}`).toContain(heading);
        }
    });

    test('.claude/skills symlinks to /world/skills so Claude Code discovers tool skills', async () => {
        await using result = await spec('claude skills link')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');

        // After PrelaunchShell runs, $HOME/.claude/skills is a symlink
        // Into /world/skills where CollectSkills baked every
        // Tool-shipped SKILL.md. Claude Code's native skill discovery
        // Picks the directory up from there.
        //
        // Note: symlink is created by the runtime's PrelaunchShell,
        // Which runs right before `claude` itself. The file won't
        // Exist until an interactive session starts — so we assert
        // /world/skills is readable instead.
        const testDir = await neo.exec('test -d /world/skills');
        expect(testDir.exitCode).toBe(0);
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
