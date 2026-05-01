import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * `spwn check` dependencies-field resolution. Parallel to the
 * tools-field tests in ../check/ — each scenario has its own frozen
 * fixture under `tests/_fixtures/check-{valid,invalid}-dep/` that
 * declares a single-agent project whose `agent.yaml` uses the
 * `dependencies:` field.
 */

describe('spwn check: dependencies field resolution', () => {
    test('accepts spwn:mempalace without error', async () => {
        // Given - a project whose neo agent declares dependencies: ["spwn:mempalace"]
        const result = await spec('deps valid').project('check-valid-dep').exec('check').run();

        // Then - check passes and does not complain about the dependency
        expect(result.exitCode).toBe(0);
        // `check` renders its report on stdout.
        expect(result.stdout.text).not.toContain('does not exist');
    });

    test('rejects nonexistent dependency refs with the same wording as tools', async () => {
        // Given - a project whose neo agent declares a bogus dependency ref
        const result = await spec('deps invalid').project('check-invalid-dep').exec('check').run();

        // Then - check fails and names the offending dependency
        expect(result.exitCode).toBe(1);
        // `check` renders its report on stdout.
        result.stdout.toContain('spwn:totally-bogus-dep');
        expect(result.stdout.text.toLowerCase()).toContain('does not exist');
    });
});
