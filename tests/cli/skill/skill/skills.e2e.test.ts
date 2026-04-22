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

    test('.claude/skills is materialised directly under the agent home', async () => {
        // Default `spwn init` scaffolds `skill:focus` into agent.yaml
        // And ships spwn/skills/focus.md; the transpile layer must
        // Lift it into /agents/<name>/.claude/skills/<skill>/SKILL.md
        // At spawn time. docker-pilot has no skills, so this test
        // Must drive a scaffold that actually declares one.
        await using result = await spec('claude skills tree')
            .project('empty')
            .exec(['init', 'up'])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        const neo = result.container('neo');

        // The transpile layer writes every resolved skill (tool-shipped
        // Plus user-authored) under /agents/<name>/.claude/skills/<skill>/SKILL.md
        // Via docker-cp at spawn time. No symlink, no /world/skills
        // Indirection — Claude Code's native walker finds the tree at
        // Its canonical location on startup.
        const dir = await neo.exec('test -d /agents/neo/.claude/skills');
        expect(dir.exitCode).toBe(0);
        expect(neo.file('/agents/neo/.claude/skills/focus/SKILL.md').exists).toBe(true);
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
