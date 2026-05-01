import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * Edge cases for world-scoped knowledge.
 *
 * Knowledge is opt-in per-world via the `worlds.<name>.knowledge` key
 * in spwn.yaml. The happy paths (mount + isolation + mixed projects)
 * are pinned in apps/cli/knowledge_mount_e2e_test.go. This file covers
 * the rough edges users hit at the fringes:
 *
 *   - a declared knowledge path that doesn't exist on disk
 *   - a project mixing a world that has knowledge with one that doesn't
 *   - a knowledge path pointing outside the project root
 *
 * These are CLI-only (no Docker) — they run fast against `spwn check`.
 */

describe('knowledge path edge cases', () => {
    test('missing knowledge directory surfaces a warning, not a crash', async () => {
        // Given - the knowledge-missing-dir fixture declares
        // `worlds.neo.knowledge: ./knowledge` in spwn.yaml but ships
        // No such directory on disk.
        // When - `spwn check` runs
        // Then - the report emits a warning naming the world and the
        // Path, plus a concrete fix (`mkdir -p ./knowledge` or drop
        // The key). Exit code is zero because this is recoverable.
        const result = await spec('knowledge missing dir')
            .project('knowledge-missing-dir')
            .exec('check')
            .run();

        expect(result.exitCode, `stdout:\n${result.stdout.text}`).toBe(0);
        expect(result.stdout.text).toMatch(/knowledge path .*does not exist/);
        expect(result.stdout.text).toContain('./knowledge');
        expect(result.stdout.text).toMatch(/mkdir -p/);
    });

    test('mixed project (one world with knowledge, one without) passes check', async () => {
        // Given - the knowledge-mixed-worlds fixture declares two
        // Worlds: `primary` has `knowledge: ./knowledge`, `ephemeral`
        // Has no knowledge key at all.
        // When - `spwn check` runs
        // Then - no errors. The info-level hint fires for the world
        // That didn't opt in, so the user can discover the switch;
        // The primary world stays silent because its knowledge is
        // Declared.
        const result = await spec('knowledge mixed worlds')
            .project('knowledge-mixed-worlds')
            .exec('check')
            .run();

        expect(result.exitCode, `stdout:\n${result.stdout.text}`).toBe(0);
        expect(result.stdout.text).toMatch(/ephemeral.*has no knowledge path/);
        // The primary world should NOT surface the info hint since
        // It explicitly opts in.
        expect(result.stdout.text).not.toMatch(/primary.*has no knowledge path/);
    });

    test('knowledge path outside project root is a warning today, not an error', async () => {
        // Given - the knowledge-outside-root fixture declares
        // `knowledge: ../escape` which resolves outside the project.
        // When - `spwn check` runs
        // Then - today's behaviour is a permissive "does not exist"
        // Warning. The manifest is not rejected. This test captures
        // Current behaviour explicitly so the day we tighten the
        // Validator to reject escaping paths, this assertion flips
        // And forces a deliberate decision.
        const result = await spec('knowledge outside root')
            .project('knowledge-outside-root')
            .exec('check')
            .run();

        // Exit code zero: today we treat this as recoverable.
        expect(result.exitCode).toBe(0);
        // The warning does mention the escaping path in some form —
        // Either as `../escape` itself or a resolved absolute path.
        // We match either shape so the test doesn't pin one renderer.
        expect(result.stdout.text).toMatch(/\.\.\/escape|does not exist/);
    });
});
