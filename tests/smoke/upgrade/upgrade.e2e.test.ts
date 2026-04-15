import { execSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

/**
 * Upgrade-cycle smoke test: build an image, mutate the project
 * (add a tool, add a skill, rewrite identity, drop in a second
 * agent), spawn again, and prove every change propagates into
 * the running container.
 *
 * This is the canary for the whole modify-rebuild-verify loop:
 *
 *   - New tool  -> content hash changes -> image rebuilt -> new
 *                  binary available inside the container.
 *   - New skill -> bind-mounted via /agents, visible instantly.
 *   - New identity body -> same bind, visible instantly.
 *   - New agent in the same world -> re-spawn picks up the new
 *     roster and the second agent home appears under /agents/.
 *
 * Every other smoke test sets SPWN_BASE_IMAGE to skip the build
 * path; this one never does. It runs serially with the rest of
 * the smoke suite because the global spwn/world:latest tag is
 * shared and parallel rebuilds would race. Docker containers are
 * tagged with a unique sh.spwn.test.run label so cleanup can
 * reap them even if the test throws mid-flight.
 */

const SPWN_BIN = resolve(import.meta.dirname, '../../../bin/spwn');
const TEST_LABEL = `smoke-upgrade-${process.pid}-${Date.now()}`;

let workdir: string;

function spwn(cmd: string, opts: { timeout?: number } = {}): string {
    return execSync(`${SPWN_BIN} ${cmd}`, {
        cwd: workdir,
        encoding: 'utf8',
        stdio: ['ignore', 'pipe', 'pipe'],
        env: { ...process.env, SPWN_TEST_LABEL: TEST_LABEL },
        timeout: opts.timeout ?? 300_000,
    });
}

function container(): string {
    // One world -> one container tagged with our test label.
    const out = execSync(`docker ps -q --filter "label=sh.spwn.test.run=${TEST_LABEL}"`, {
        encoding: 'utf8',
    }).trim();
    if (!out) {
        throw new Error(`no container found for test label ${TEST_LABEL}`);
    }
    return out.split('\n')[0];
}

function exec(cid: string, shellCmd: string): string {
    return execSync(`docker exec ${cid} sh -c ${JSON.stringify(shellCmd)}`, {
        encoding: 'utf8',
    }).trim();
}

function imageVersion(): string {
    return execSync(
        `docker inspect spwn/world:latest --format '{{index .Config.Labels "sh.spwn.image-version"}}'`,
        { encoding: 'utf8' },
    ).trim();
}

beforeAll(() => {
    workdir = mkdtempSync(join(tmpdir(), 'spwn-smoke-upgrade-'));
});

afterAll(() => {
    // Best-effort cleanup. Never throw here or we mask the real
    // Failure in the test body.
    try {
        spwn('down');
    } catch {
        /* Ignore */
    }
    try {
        execSync(
            `docker ps -aq --filter "label=sh.spwn.test.run=${TEST_LABEL}" | xargs -r docker rm -f`,
            { stdio: 'ignore', shell: '/bin/sh' },
        );
    } catch {
        /* Ignore */
    }
    if (workdir) {
        rmSync(workdir, { recursive: true, force: true });
    }
});

describe('smoke: project upgrade cycle', () => {
    test('modify tools / skills / identity / roster and re-spawn propagates everything', () => {
        // ── Phase 1: baseline scaffold + spawn ───────────────────

        spwn('init');
        spwn('up', { timeout: 600_000 });

        const baselineVersion = imageVersion();
        expect(baselineVersion).not.toBe('');

        const cid1 = container();

        // Baseline tools from the default scaffold (unix + git + python).
        expect(exec(cid1, 'command -v python3')).not.toBe('');
        expect(exec(cid1, 'command -v git')).not.toBe('');
        // The Phase 2 marker tool (@spwn/qmd) is added in Phase 2
        // and verified in Phase 3. Asserting its absence here would
        // be brittle against evolving base images, so we rely on
        // the image-version-label diff + the Phase 3 presence
        // check to prove the rebuild.

        // Baseline: one agent home at /agents/neo, no trinity yet.
        expect(exec(cid1, 'test -d /agents/neo && echo ok')).toBe('ok');
        expect(() => exec(cid1, 'test -d /agents/trinity && echo ok')).toThrow();

        spwn('down');

        // ── Phase 2: mutate the project ──────────────────────────

        // 2a. Add a tool the default scaffold doesn't ship. The
        //     @spwn/qmd pack runs `npm install -g @tobilu/qmd`,
        //     which adds a new RUN layer to the Dockerfile - the
        //     content hash changes and the cache misses.
        writeFileSync(
            join(workdir, 'spwn/agents/neo/agent.yaml'),
            [
                'name: neo',
                '',
                'runtime:',
                '  backend: "@spwn/claude-code"',
                '',
                'tools:',
                '  - "@spwn/unix"',
                '  - "@spwn/git"',
                '  - "@spwn/python"',
                '  - "@spwn/qmd"',
                '',
            ].join('\n'),
        );

        // 2b. Drop in a new skill file. Skills live under the
        //     Shared /agents bind mount, so the new file becomes
        //     Visible inside the container as soon as it lands on
        //     Disk - no rebuild required. We verify it's present
        //     After the second spawn anyway to rule out a stale
        //     Cache holding old contents.
        writeFileSync(
            join(workdir, 'spwn/agents/neo/skills/arithmetic.md'),
            '# Arithmetic\n\nReturn 2 + 2 when asked.\n',
        );

        // 2c. Rewrite the agent's identity. Same story as skills:
        //     Lives under the /agents bind, so the container sees
        //     The new bytes after the rewrite.
        writeFileSync(
            join(workdir, 'spwn/agents/neo/identity/profile.md'),
            '# Upgraded identity\n\nI am the post-upgrade neo.\n',
        );

        // 2d. Add a second agent to the same world. A new
        //     Directory tree under spwn/agents/trinity/ + a new
        //     Entry in spwn.yaml's worlds map. The next `spwn up`
        //     Brings up both homes inside the shared container.
        const trinityDir = join(workdir, 'spwn/agents/trinity');
        mkdirSync(join(trinityDir, 'identity'), { recursive: true });
        mkdirSync(join(trinityDir, 'skills'), { recursive: true });
        mkdirSync(join(trinityDir, 'knowledge'), { recursive: true });
        mkdirSync(join(trinityDir, 'playbooks'), { recursive: true });
        mkdirSync(join(trinityDir, 'journal'), { recursive: true });
        writeFileSync(
            join(trinityDir, 'agent.yaml'),
            [
                'name: trinity',
                '',
                'runtime:',
                '  backend: "@spwn/claude-code"',
                '',
                'tools:',
                '  - "@spwn/unix"',
                '',
            ].join('\n'),
        );
        writeFileSync(
            join(trinityDir, 'AGENT.md'),
            "# trinity\n\nYou are trinity, neo's partner.\n",
        );
        writeFileSync(
            join(trinityDir, 'identity/profile.md'),
            '# trinity\n\nOperator, partner, edge case specialist.\n',
        );

        // Rewrite spwn.yaml to list both agents under the neo world.
        writeFileSync(
            join(workdir, 'spwn.yaml'),
            [
                'version: 2',
                `name: smoke-upgrade`,
                '',
                'worlds:',
                '  neo:',
                '    agents: [neo, trinity]',
                '    workspaces: [.]',
                '',
            ].join('\n'),
        );

        // ── Phase 3: re-spawn, verify every change propagated ────

        spwn('up', { timeout: 600_000 });

        const upgradedVersion = imageVersion();
        expect(upgradedVersion).not.toBe(baselineVersion);

        const cid2 = container();

        // New tool: @spwn/qmd installed a `qmd` binary via npm
        // during the content-hash-triggered rebuild. If this
        // command fails, either the rebuild didn't happen or the
        // generator dropped the tool - both are regressions in
        // the cache / generator pipeline.
        expect(exec(cid2, 'command -v qmd')).not.toBe('');

        // New skill: present inside neo's home at the expected
        // Path, with the content we wrote on the host.
        expect(exec(cid2, 'cat /agents/neo/skills/arithmetic.md')).toContain(
            'Return 2 + 2 when asked',
        );

        // New identity: the upgraded body is visible.
        expect(exec(cid2, 'cat /agents/neo/identity/profile.md')).toContain('post-upgrade neo');

        // Both agent homes exist; the second agent came along for
        // The ride as part of the same world spawn.
        expect(exec(cid2, 'test -d /agents/neo && echo ok')).toBe('ok');
        expect(exec(cid2, 'test -d /agents/trinity && echo ok')).toBe('ok');
        expect(exec(cid2, 'cat /agents/trinity/identity/profile.md')).toContain(
            'edge case specialist',
        );

        // Old baseline tools are still there too - a new tool
        // Must not regress the existing ones.
        expect(exec(cid2, 'command -v python3')).not.toBe('');
        expect(exec(cid2, 'command -v git')).not.toBe('');

        spwn('down');
    });
});
