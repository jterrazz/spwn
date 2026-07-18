import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * System skills / agent prompt injection. The per-agent CLAUDE.md is the
 * single source of ambient context — it inlines physics, faculties,
 * roster, conventions, and the playbook index. Tool-shipped skills are
 * materialised under the agent home's `.claude/skills/<skill>/SKILL.md` at
 * spawn time. The first two specs spin a real world (docker-backed); the
 * last is a CLI-only project-scoped `skill new`. Every result binds with
 * `await using` (rule B5).
 */
describe('system skills infrastructure (docker)', () => {
    test('per-agent CLAUDE.md is laid down with world context inlined', async () => {
        // Given - a docker-pilot world brought online
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        // Then - neo's CLAUDE.md exists, is non-trivial, and carries every standard section header
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Created container');
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        const claude = neo.file('/agents/neo/CLAUDE.md');
        expect(claude.exists).toBe(true);
        expect(claude.content.length).toBeGreaterThan(200);
        expect(claude.content).toContain('neo');
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
        // Given - an empty project scaffolded with a claude-code backend then brought up (docker-pilot ships no skills)
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init --backend claude-code', 'up']);

        // Then - the transpile layer writes the resolved skill under the agent's native .claude/skills tree via docker-cp
        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        const neo = result.container('neo');
        const dir = await neo.exec('test -d /agents/neo/.claude/skills');
        expect(dir.exitCode).toBe(0);
        expect(neo.file('/agents/neo/.claude/skills/focus/SKILL.md').exists).toBe(true);
    });
});

describe('spwn skill new (project-local)', () => {
    test('skill new inside a project writes into the project tree', async () => {
        // Given - an initialised empty project with a new skill authored
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'skill new my-skill']);

        // Then - it lands under spwn/skills/ (not ~/.spwn/skills/)
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/skills/my-skill.md').exists).toBe(true);
    });
});
