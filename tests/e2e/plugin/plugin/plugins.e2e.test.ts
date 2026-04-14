import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn check` plugin-field resolution. Parallel to the tools-field
 * tests in ../check/ — each scenario has its own frozen fixture under
 * `tests/fixtures/check-{valid,invalid}-plugin/` that declares a
 * single-agent project whose `agent.yaml` uses the `plugins:` field.
 */

describe('spwn check: plugins field resolution', () => {
    test('accepts @spwn/mempalace without error', async () => {
        // GIVEN - a project whose neo agent declares plugins: ["@spwn/mempalace"]
        const result = await spec('plugins valid')
            .project('check-valid-plugin')
            .exec('check')
            .run();

        // THEN - check passes and does not complain about the plugin
        expect(result.exitCode).toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).not.toContain('does not exist');
    });

    test('rejects nonexistent plugin refs with the same wording as tools', async () => {
        // GIVEN - a project whose neo agent declares a bogus plugin
        const result = await spec('plugins invalid')
            .project('check-invalid-plugin')
            .exec('check')
            .run();

        // THEN - check fails and names the offending plugin
        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).toContain('@spwn/totally-bogus-plugin');
        expect(combined.toLowerCase()).toContain('does not exist');
    });
});
