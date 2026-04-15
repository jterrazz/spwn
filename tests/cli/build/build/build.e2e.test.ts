import { execSync } from 'node:child_process';
import { afterEach, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn build` e2e coverage. Each happy-path test produces a real
 * Docker image derived from the pre-built `spwn-test:latest` base
 * image, so we scope cleanup to the `sh.spwn.kind=project-build`
 * label after each test to avoid cruft accumulating across runs.
 *
 * The cleanup label is distinct from sh.spwn.kind=world (containers)
 * and sh.spwn.kind=architect (daemon), so we never touch running
 * worlds here.
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
        const result = await spec('build no project').project('empty').exec('build').run();

        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('spwn init');
        expect(stderr).toContain('spwn.yaml');
    });

    test('produces a tagged image from docker-pilot', async () => {
        const result = await spec('build basic')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build')
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.stderr.text).toContain('Built image');

        const images = execSync('docker images --format "{{.Repository}}:{{.Tag}}"', {
            encoding: 'utf8',
        });
        expect(images).toMatch(/spwn-docker-pilot:latest/);
    });

    test('--json emits a machine-readable report', async () => {
        const result = await spec('build json')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build --json')
            .run();

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

    test('--tag uses a custom image tag', async () => {
        const result = await spec('build custom tag')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build --tag spwn-test-qa:v1')
            .run();

        expect(result.exitCode).toBe(0);
        const images = execSync('docker images --format "{{.Repository}}:{{.Tag}}"', {
            encoding: 'utf8',
        });
        expect(images).toMatch(/spwn-test-qa:v1/);
    });

    test('catches project validation errors before touching Docker', async () => {
        // Given - a project whose agent references a nonexistent tool
        const result = await spec('build broken')
            .project('check-invalid-tool')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('build')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('spwn check');
        // The error should come from the validation layer, not Docker.
        expect(result.stderr.text.toLowerCase()).not.toContain('docker build');
    });
});
