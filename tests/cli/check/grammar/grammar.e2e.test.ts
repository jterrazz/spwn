import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Grammar-contract suite for dependency refs.
 *
 * The CLI accepts bare names as shorthand (`spwn init qmd`,
 * `spwn install qmd`) and auto-promotes them to `spwn:<name>`
 * via the catalog resolver. Manifests stay strict: every dep in
 * agent.yaml must use one of the five explicit schemes (spwn:,
 * github:, skill:, tool:, hook:). Bare names, the retired `local:`
 * alias, and the legacy `@owner/name` shape are all rejected.
 *
 * This file locks the boundary: every rejected shape surfaces as a
 * `spwn check` error, and the CLI → manifest path consistently
 * writes the canonical scheme-form even when the user typed a bare
 * name.
 */

describe('dependency grammar', () => {
    test('spwn check rejects bare / local: / @owner/ refs in agent.yaml', async () => {
        // Given - the check-legacy-refs fixture ships an agent.yaml
        // Whose `dependencies:` list mixes three rejected shapes:
        //   - "python"        (bare, no scheme)
        //   - "local:foo"     (retired alias)
        //   - "@acme/foo"     (legacy @owner/name)
        // When - `spwn check` runs over the project
        // Then - every entry surfaces as a distinct error naming the
        // Offending ref, all on the same agent.yaml#deps location,
        // And the exit code is non-zero so scripts notice.
        const result = await spec('grammar rejects bad refs')
            .project('check-legacy-refs')
            .exec('check')
            .run();

        expect(result.exitCode).not.toBe(0);

        const report = result.stdout.text;
        // Each rejected ref produces its own error line — users
        // Should see exactly which entries are malformed.
        expect(report).toContain('dependency "python" is invalid');
        expect(report).toContain('dependency "local:foo" is invalid');
        expect(report).toContain('dependency "@acme/foo" is invalid');

        // Every hint must point at the five-scheme grammar so the
        // User has a concrete fix path.
        expect(report).toMatch(/skill:<name>/);
        expect(report).toMatch(/tool:<name>/);
        expect(report).toMatch(/hook:<name>/);
        expect(report).toMatch(/spwn:<name>/);
        expect(report).toMatch(/github:<owner>\/<repo>/);
    });

    test('bare names accepted on the CLI land as spwn:<name> in agent.yaml', async () => {
        // This is the keystone of the sugar mechanism: the CLI
        // Resolver auto-promotes bare names, but the resolved form is
        // What reaches disk. A follow-up `spwn check` must then pass
        // Because the manifest carries the canonical scheme-form.
        const result = await spec('bare on cli yields spwn: on disk')
            .project('empty')
            .exec(['init', 'install python', 'check'])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);

        const manifest = result.file('spwn/agents/neo/agent.yaml').content;
        // The bare input `python` was canonicalised on write. No
        // Manifest ever sees the bare token.
        expect(manifest).toContain('spwn:python');
        expect(manifest).not.toMatch(/^\s*-\s+python\s*$/m);
    });

    test('manifest is the strict boundary even when the CLI would accept', async () => {
        // Given - a project scaffolded by init (clean state), we run
        // Several install invocations mixing bare, catalog-explicit,
        // And local-explicit refs. The resolver must canonicalise
        // Every input to its scheme-form before it lands on disk —
        // The on-disk shape is what `spwn check` validates.
        //
        // To exercise the boundary directly, we also prove the
        // Inverse: a hand-edited agent.yaml with a bare name (the
        // Check-legacy-refs fixture at the top of this file) must be
        // Rejected. The CLI itself, though, never produces such a
        // Manifest.
        // skill:focus is scaffolded by `spwn init`, so it's available
        // to install — prior revisions of this test used
        // `skill:paper-reading` which would require a separate
        // `spwn skill new` call. The resolver now errors on missing
        // local files at install time, which is the right behaviour
        // but broke the old form.
        const result = await spec('cli canonicalises mixed input')
            .project('empty')
            .exec([
                'init',
                'install qmd',
                'install skill:focus --agent neo',
                'install spwn:unix',
            ])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);

        const manifest = result.file('spwn/agents/neo/agent.yaml').content;
        // Every entry on disk carries an explicit scheme — bare
        // `qmd` becomes `spwn:qmd`; the explicit `skill:` and
        // `spwn:` pass through unchanged.
        expect(manifest).toContain('spwn:qmd');
        expect(manifest).toContain('skill:focus');
        expect(manifest).toContain('spwn:unix');
        // Nothing on disk is bare — a simple structural scan proves
        // The grammar is preserved by the CLI writer.
        const bareHits = manifest.match(/^\s*-\s+[a-z0-9][a-z0-9-]*\s*$/gm);
        expect(bareHits, `manifest carries bare entries: ${bareHits?.join(', ')}`).toBeNull();
    });
});
