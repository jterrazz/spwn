import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot judge JSON by its structure: each report is compared whole as the value it parses to

describe('json-report', () => {
    test('a valid project reports no issue', async () => {
        // Given - the single-agent project
        await using result = await cli
            .fixture(directory('../__fixtures__/single-agent/'))
            .exec('check --json');

        // Then - the envelope is valid and empty
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            issues: [],
            manifestPath: '{{workdir}}/spwn.yaml',
            summary: { errors: 0, info: 0, warnings: 0 },
            valid: true,
        });
    });

    test('an unknown built-in is one manifest error naming every built-in', async () => {
        // Given - a project whose agent depends on spwn:nonexistent
        await using result = await cli
            .fixture(directory('../__fixtures__/check-invalid-tool/'))
            .exec('check --json');

        // Then - the envelope carries the one error and its hint
        expect(result.exitCode).toBe(1);
        expect(result.json).toMatch({
            issues: [
                {
                    hint: 'available built-ins: spwn:architect, spwn:build, spwn:cli, spwn:docker-cli, spwn:gh, spwn:git, spwn:mcp2cli, spwn:mempalace, spwn:node, spwn:notion, spwn:python, spwn:qmd, spwn:unix, spwn:x, spwn:claude-code, spwn:codex',
                    level: 'error',
                    message: 'dependency "spwn:nonexistent" does not exist',
                    path: 'spwn/agents/neo/agent.yaml#deps',
                    source: 'manifest',
                },
            ],
            manifestPath: '{{workdir}}/spwn.yaml',
            summary: { errors: 1, info: 0, warnings: 0 },
            valid: false,
        });
    });

    test('--deep reports an empty agent prompt as a compile issue', async () => {
        // Given - the single-agent project with an empty AGENTS.md, checked deep
        await using result = await cli
            .fixture(directory('../__fixtures__/single-agent/'))
            .fixture(directory('../__fixtures__/empty-agents-md/'))
            .exec('check --deep --json');

        // Then - the envelope carries the compile issue
        expect(result.exitCode).toBe(1);
        expect(result.json).toMatch({
            valid: false,
            manifestPath: '{{workdir}}/spwn.yaml',
            summary: {
                errors: 1,
                warnings: 0,
                info: 0,
            },
            issues: [
                {
                    level: 'error',
                    path: 'spwn/agents/neo/AGENTS.md',
                    message: 'agent prompt is missing or empty',
                    hint: "create spwn/agents/neo/AGENTS.md with the agent's system prompt",
                    source: 'compile',
                },
            ],
        });
    });
});
