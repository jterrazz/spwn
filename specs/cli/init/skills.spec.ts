import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read a container back: the skills tree is listed inside the running world

describe('skills', () => {
    test('a claude-code agent finds its resolved skills under its own .claude/skills', async () => {
        // Given - a project scaffolded for claude-code, then brought up
        await using result = await cli.exec(['init --backend claude-code', 'up']);

        // Then - the starter skill sits in the agent's native skills tree inside the container
        expect(result.exitCode).toBe(0);
        const tree = await result
            .container('neo')
            .exec('cd /agents/neo/.claude/skills && find . -type f | sort');
        expect(tree.stdout.text).toBe('./focus/SKILL.md\n');
    });
});
