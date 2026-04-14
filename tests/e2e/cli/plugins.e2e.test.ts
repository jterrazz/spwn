import { spawnSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { describe, expect, test } from 'vitest';

import { stripAnsi } from '../../setup/output-helpers.js';

const SPWN_BIN = resolve(import.meta.dirname, '../../../bin/spwn');

/**
 * Build a minimal spwn project whose only agent (`neo`) declares the
 * given plugin list in its agent.yaml.
 */
function makeProject(plugins: string[]): string {
    const root = mkdtempSync(join(tmpdir(), 'spwn-plugins-test-'));

    writeFileSync(
        join(root, 'spwn.yaml'),
        `version: 2
name: plugins-test
worlds:
  default:
    agents: [neo]
    workspaces: [.]
`,
    );

    const agentDir = join(root, 'spwn', 'agents', 'neo');
    mkdirSync(join(agentDir, 'core'), { recursive: true });
    mkdirSync(join(agentDir, 'skills'), { recursive: true });
    mkdirSync(join(agentDir, 'knowledge'), { recursive: true });
    mkdirSync(join(agentDir, 'playbooks'), { recursive: true });
    mkdirSync(join(agentDir, 'journal'), { recursive: true });
    writeFileSync(join(agentDir, 'core', 'profile.md'), '# neo\n\nTest agent.\n');
    writeFileSync(join(agentDir, 'CLAUDE.md'), '# neo\n\n@core/profile.md\n');

    const pluginLines = plugins.map((p) => `  - "${p}"`).join('\n');
    writeFileSync(
        join(agentDir, 'agent.yaml'),
        `name: neo
runtime:
  backend: "@spwn/claude-code"
plugins:
${pluginLines}
`,
    );

    return root;
}

function spwnCheck(projectRoot: string): { exitCode: number; output: string } {
    const result = spawnSync(SPWN_BIN, ['check'], {
        cwd: projectRoot,
        encoding: 'utf8',
        env: { ...process.env, INIT_CWD: undefined } as NodeJS.ProcessEnv,
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout: 15_000,
    });
    return {
        exitCode: result.status ?? 1,
        output: stripAnsi((result.stdout ?? '') + (result.stderr ?? '')),
    };
}

describe('spwn check: plugins field resolution', () => {
    test('accepts @spwn/mempalace without error', () => {
        const root = makeProject(['@spwn/mempalace']);

        const result = spwnCheck(root);
        expect(result.exitCode, `output:\n${result.output}`).toBe(0);
        expect(result.output).not.toContain('does not exist');
    });

    test('rejects nonexistent plugin refs with the same wording as tools', () => {
        const root = makeProject(['@spwn/totally-bogus-plugin']);

        const result = spwnCheck(root);
        expect(result.exitCode).not.toBe(0);
        expect(result.output).toContain('@spwn/totally-bogus-plugin');
        expect(result.output.toLowerCase()).toContain('does not exist');
    });
});
