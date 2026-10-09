import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot judge JSON by its structure nor compare a directory tree whole

describe('tree-only', () => {
    test('--json reports every file of the compiled tree', async () => {
        // Given - the docker-pilot project compiled with --json
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec('build --tree-only --json');

        // Then - the report lists the tree whole
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            runtime: 'claude-code',
            world: 'neo',
            treeFiles: 2,
            treeOnly: true,
            outDir: '{{workdir}}/dist',
            paths: ['agents/neo/.claude/settings.json', 'agents/neo/CLAUDE.md'],
        });
    });

    test('a codex project compiles to AGENTS.md, the codex config and native skills, and nothing of Claude Code', async () => {
        // Given - the codex-pilot project compiled with --json
        await using result = await cli
            .fixture(directory('../__fixtures__/codex-pilot/'))
            .exec('build --tree-only --json');

        // Then - the report lists the tree whole, and the agent's home holds the codex files as written
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            runtime: 'codex',
            world: 'neo',
            treeFiles: 3,
            treeOnly: true,
            outDir: '{{workdir}}/dist',
            paths: [
                'agents/neo/.agents/skills/focus/SKILL.md',
                'agents/neo/.codex/config.toml',
                'agents/neo/AGENTS.md',
            ],
        });
        await expect(result.directory('dist/agents/neo')).toMatch(
            directory('./__expected__/codex-home/'),
        );
    });
});
