import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn check` pack-field resolution. Parallel to the tools-field
 * tests in ../check/ — each scenario has its own frozen fixture under
 * `tests/fixtures/check-{valid,invalid}-pack/` that declares a
 * single-agent project whose `agent.yaml` uses the `deps:` field.
 */

describe('spwn check: deps field resolution', () => {
    test('accepts @spwn/mempalace without error', async () => {
        // Given - a project whose neo agent declares deps: ["@spwn/mempalace"]
        const result = await spec('packs valid').project('check-valid-pack').exec('check').run();

        // Then - check passes and does not complain about the pack
        expect(result.exitCode).toBe(0);
        // `check` renders its report on stdout.
        expect(result.stdout.text).not.toContain('does not exist');
    });

    test('rejects nonexistent pack refs with the same wording as tools', async () => {
        // Given - a project whose neo agent declares a bogus pack ref
        const result = await spec('packs invalid')
            .project('check-invalid-pack')
            .exec('check')
            .run();

        // Then - check fails and names the offending pack
        expect(result.exitCode).toBe(1);
        // `check` renders its report on stdout.
        result.stdout.toContain('@spwn/totally-bogus-pack');
        expect(result.stdout.text.toLowerCase()).toContain('does not exist');
    });
});
