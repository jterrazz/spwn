import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Tool reference classification by `spwn check`:
 *
 *   spwn:<name>              → built-in tool dependency (must exist)
 *   github:<owner>/<repo>    → remote registry (not yet supported)
 *   skill:<name>             → local skill (spwn/skills/<name>.md)
 *   tool:<name>              → local tool (spwn/tools/<name>/)
 *   hook:<name>              → local hook (spwn/hooks/<name>.sh)
 *
 * The canonical success path is already covered by the check suite via
 * the single-agent fixture. These tests pin the classifier edges: a
 * mixed spwn: + tool/ project should pass, a registry ref should fail
 * with explicit wording quoting the offending ref.
 */

describe('spwn check: tool ref classification', () => {
    test('accepts mixed spwn: and tool/ refs without error', async () => {
        // Given - the mixed-tool-refs fixture declares spwn:unix plus
        // A local tool `tool/my-local-tool` that actually exists on disk
        // Under spwn/tools/.
        const result = await spec('mixed refs').project('mixed-tool-refs').exec('check').run();

        // Then - check passes and neither the registry nor the
        // "does not exist" errors are raised.
        expect(result.exitCode, `output:\n${result.stdout.text}`).toBe(0);
        // `check` writes its report to stdout.
        expect(result.stdout.text).not.toContain('remote registries are not yet supported');
        expect(result.stdout.text).not.toContain('does not exist');
    });

    test('rejects github:<owner>/<repo> registry refs with explicit wording', async () => {
        // Given - check-registry-tool has dependencies: ["github:jterrazz/foo"]
        const result = await spec('registry ref')
            .project('check-registry-tool')
            .exec('check')
            .run();

        // Then - exits non-zero and the error quotes the offending ref
        expect(result.exitCode).toBe(1);
        // `check` writes its report (errors included) to stdout.
        result.stdout.toContain('remote registries are not yet supported');
        result.stdout.toContain('github:jterrazz/foo');
    });
});
