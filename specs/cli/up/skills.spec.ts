import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read a container back: the skills tree is listed inside the running world

describe('skills', () => {
    test('a claude-code agent finds its resolved skills under its own .claude/skills', async () => {
        // Given - the scaffold with neo pinned to claude-code, brought up
        await using result = await cli
            .fixture(directory('./__fixtures__/scaffolded-project/'))
            .fixture(directory('./__fixtures__/claude-code-neo/'))
            .exec('up');

        // Then - the starter skill sits in the agent's native skills tree inside the container
        expect(result.exitCode).toBe(0);
        const tree = await result
            .container('neo')
            .exec('cd /agents/neo/.claude/skills && find . -type f | sort');
        expect(tree.stdout.text).toBe('./focus/SKILL.md\n');
    });
});
