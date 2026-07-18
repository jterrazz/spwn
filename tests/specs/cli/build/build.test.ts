import { execSync } from 'node:child_process';
import { afterEach, describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn build` — produces a real Docker image derived from the pre-built
 * `spwn-test:latest` base. Cleanup is scoped to the
 * `sh.spwn.kind=project-build` label after each test to avoid image cruft
 * accumulating across runs (distinct from world containers and the
 * architect daemon, so running worlds are never touched). Image
 * introspection stays on raw `docker`/`execSync` — a built image is not a
 * spawned container, so `result.container(...)` cannot reach it. The
 * runner is docker-aware, so every result binds with `await using` (B5).
 */
describe('spwn build', () => {
    afterEach(() => {
        try {
            const ids = execSync('docker images -q --filter label=sh.spwn.kind=project-build', {
                encoding: 'utf8',
                timeout: 10_000,
            })
                .trim()
                .split('\n')
                .filter(Boolean);
            if (ids.length > 0) {
                execSync(`docker rmi -f ${ids.join(' ')}`, { stdio: 'ignore' });
            }
        } catch {
            // Nothing to clean — ignore.
        }
    });

    test('errors when run outside a spwn project', async () => {
        // Given - the empty fixture has no spwn.yaml anywhere up the tree
        await using result = await cli.fixture('$FIXTURES/empty/').exec('build');

        // Then - exits non-zero nudging the user at spwn init (scalpel: stderr lowercased for a case-insensitive probe)
        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('spwn init');
        expect(stderr).toContain('spwn.yaml');
    });

    test('produces a tagged image from docker-pilot', async () => {
        // Given - the docker-pilot fixture built against the test base image
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build');

        // Then - build succeeds and the tagged image lands in docker (scalpel: image list is genuine docker plumbing)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Built image');
        const images = execSync('docker images --format "{{.Repository}}:{{.Tag}}"', {
            encoding: 'utf8',
        });
        expect(images).toMatch(/spwn-docker-pilot:latest/);
    });

    test('--json emits a machine-readable report', async () => {
        // Given - the docker-pilot fixture built with --json
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build --json');

        // Then - the report carries stable fields (scalpel: imageId sha + treeFiles count are dynamic)
        expect(result.exitCode).toBe(0);
        const report = result.json.value as {
            baseImage: string;
            imageId: string;
            runtime: string;
            tag: string;
            treeFiles: number;
        };
        expect(report.runtime).toBe('claude-code');
        expect(report.tag).toBe('spwn-docker-pilot:latest');
        expect(report.imageId).toMatch(/^sha256:[a-f0-9]{12}/);
        expect(report.baseImage).toBe('spwn-test:latest');
        expect(report.treeFiles).toBeGreaterThan(0);
    });

    test('codex image contains AGENTS.md, .codex config, and native skills', async () => {
        // Given - the codex-pilot fixture built with a custom tag
        await using result = await cli
            .fixture('$FIXTURES/codex-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build --tag spwn-codex-pilot:test');

        // Then - the image internals follow codex conventions (scalpel: inspecting a built image needs a docker run)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('runtime=codex');
        const check = execSync(
            [
                'docker run --rm --entrypoint sh spwn-codex-pilot:test -lc',
                JSON.stringify(
                    [
                        'test -f /world/agents/neo/AGENTS.md',
                        'test -f /world/agents/neo/.codex/config.toml',
                        'test -f /world/agents/neo/.agents/skills/focus/SKILL.md',
                        'test ! -e /world/agents/neo/CLAUDE.md',
                        'test ! -e /world/agents/neo/.claude',
                        'test ! -e /world/agents/neo/.codex/skills',
                        'grep -q "Codex pilot prompt" /world/agents/neo/AGENTS.md',
                        String.raw`grep -q "model = \"gpt-5\"" /world/agents/neo/.codex/config.toml`,
                        'grep -q "Focus Skill" /world/agents/neo/.agents/skills/focus/SKILL.md',
                        'echo ok',
                    ].join(' && '),
                ),
            ].join(' '),
            { encoding: 'utf8', timeout: 30_000 },
        );
        expect(check.trim()).toBe('ok');
    });

    test('--tag uses a custom image tag', async () => {
        // Given - the docker-pilot fixture built with an explicit tag
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build --tag spwn-test-qa:v1');

        // Then - the custom tag lands in docker (scalpel: image list is genuine docker plumbing)
        expect(result.exitCode).toBe(0);
        const images = execSync('docker images --format "{{.Repository}}:{{.Tag}}"', {
            encoding: 'utf8',
        });
        expect(images).toMatch(/spwn-test-qa:v1/);
    });

    test('catches project validation errors before touching Docker', async () => {
        // Given - a project whose agent references a nonexistent tool
        await using result = await cli
            .fixture('$FIXTURES/check-invalid-tool/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build');

        // Then - the error comes from validation, not docker (scalpel: presence + absence probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('spwn check');
        expect(result.stderr.text.toLowerCase()).not.toContain('docker build');
    });
});
