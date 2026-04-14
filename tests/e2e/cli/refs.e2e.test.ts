import { spawnSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { describe, expect, test } from 'vitest';

import { stripAnsi } from '../../setup/output-helpers.js';

const SPWN_BIN = resolve(import.meta.dirname, '../../../bin/spwn');

/**
 * Build a minimal spwn project on disk at a fresh temp dir and return
 * its absolute path. The project has one agent (`neo`) whose
 * `agent.yaml` declares the given tool list, plus a single default
 * world that deploys neo.
 */
function makeProject(tools: string[]): string {
    const root = mkdtempSync(join(tmpdir(), 'spwn-refs-test-'));

    writeFileSync(
        join(root, 'spwn.yaml'),
        `version: 2
name: refs-test
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

    const toolLines = tools.map((t) => `  - "${t}"`).join('\n');
    writeFileSync(
        join(agentDir, 'agent.yaml'),
        `name: neo
runtime:
  backend: "@spwn/claude-code"
tools:
${toolLines}
`,
    );

    return root;
}

/**
 * Invoke `spwn check` in the given project root. The check command
 * walks up from cwd looking for spwn.yaml, so we must spawn with an
 * explicit cwd rather than going through the default helpers.
 */
function spwnCheck(projectRoot: string): {
    exitCode: number;
    output: string;
} {
    const result = spawnSync(SPWN_BIN, ['check'], {
        cwd: projectRoot,
        encoding: 'utf8',
        env: { ...process.env, INIT_CWD: undefined } as NodeJS.ProcessEnv,
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout: 15_000,
    });
    const stdout = result.stdout ?? '';
    const stderr = result.stderr ?? '';
    return {
        exitCode: result.status ?? 1,
        output: stripAnsi(stdout + stderr),
    };
}

describe('spwn check: tool ref classification', () => {
    test('accepts mixed @spwn and local refs without error', () => {
        const root = makeProject(['@spwn/unix', 'my-local-tool']);
        // Create the local pack on disk so it resolves to RefLocal.
        mkdirSync(join(root, 'spwn', 'tools', 'my-local-tool'), { recursive: true });

        const result = spwnCheck(root);
        expect(result.exitCode, `output:\n${result.output}`).toBe(0);
        expect(result.output).not.toContain('remote registries are not yet supported');
        expect(result.output).not.toContain('does not exist');
    });

    test('rejects @<owner>/<name> registry refs with explicit wording', () => {
        const root = makeProject(['@spwn/unix', '@jterrazz/python']);

        const result = spwnCheck(root);
        expect(result.exitCode).not.toBe(0);
        expect(result.output).toContain('remote registries are not yet supported');
        // The error must quote the offending ref so the user knows which line to fix.
        expect(result.output).toContain('@jterrazz/python');
    });
});
