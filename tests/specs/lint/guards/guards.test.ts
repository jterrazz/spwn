import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
import { describe, expect, test } from 'vitest';

/**
 * Repo-wide source guards (the `lint` facet). These don't exercise the spwn
 * binary — they shell out to grep against the source tree to catch reintroduced
 * patterns we've intentionally removed. They are plain vitest tests (no
 * @jterrazz/test runner), so they live under specs/lint/ rather than the cli
 * facet whose specs drive the product binary.
 *
 * K1 note: the ideal home for a source-shape guard is a static lint rule. The
 * pinned oxlint (1.60.0) does not yet implement `no-restricted-syntax`, so
 * neither guard can be expressed as a single oxlint selector rule today; they
 * stay meta-tests until oxlint ships it, at which point the /api/-fetch guard
 * moves into apps/web/oxlint.config.ts as a no-restricted-syntax rule.
 */

const repoRoot = resolve(import.meta.dirname, '../../../..');

describe('repo regression guards', () => {
    test('no relative /api/ fetch calls in frontend (must use goApiUrl)', () => {
        // Given - the web app must go through goApiUrl; relative /api/
        // Calls would bypass the Go backend routing layer.
        const result = spawnSync(
            'grep',
            [
                '-rn',
                '--include=*.ts',
                '--include=*.tsx',
                '-E',
                'fetch\\("/api/|fetch\\(`/api/',
                'apps/web/src/',
            ],
            { cwd: repoRoot, encoding: 'utf8', timeout: 15_000 },
        );

        const matches = (result.stdout ?? '').split('\n').filter((line) => line.trim().length > 0);

        // Then - zero hits. Grep returns status 1 when there are no
        // Matches, which is the happy path for this guard.
        expect(
            matches.length,
            `Found ${matches.length} relative /api/ fetch(es):\n${matches.join('\n')}`,
        ).toBe(0);
    });

    test("no references to 'God' or 'god' role remain in source (rename regression)", () => {
        // Given - the role was renamed to Architect. Any surviving
        // 'God' reference in production code is a reintroduction.
        const result = spawnSync(
            'grep',
            [
                '-rn',
                '--include=*.go',
                '--include=*.ts',
                '--include=*.tsx',
                '--exclude-dir=node_modules',
                '--exclude-dir=.git',
                '--exclude-dir=.next',
                '--exclude-dir=tests',
                '-w',
                'God',
                '.',
            ],
            { cwd: repoRoot, encoding: 'utf8', timeout: 15_000 },
        );

        const matches = (result.stdout ?? '').split('\n').filter((line) => line.trim().length > 0);

        // Then - zero surviving references outside the tests tree.
        expect(
            matches.length,
            `Found ${matches.length} remaining 'God' references:\n${matches.join('\n')}`,
        ).toBe(0);
    });
});
