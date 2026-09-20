import { expect, test } from 'vitest';

import { required } from '../../_support/required.js';
import { cli } from '../cli.specification.js';

/**
 * `spwn check --json` — the machine-readable report. These stay in code
 * because the comparison is STRUCTURAL: `expect(result.json).toMatch(…)`
 * judges the envelope by shape against `_expected/*.json`, and the compile
 * pass is filtered by a field rather than read as text. A document's
 * `stdout:` is byte-exact prose; it has no vocabulary for either. The text
 * reports next door are documents.
 *
 * The runner is docker-aware, so every result binds with `await using` even
 * though check spawns no containers (rule B5).
 */

test('emits a JSON report for a valid project', async () => {
    // Given - the frozen single-agent fixture
    await using result = await cli.fixture('$FIXTURES/single-agent/').exec('check --json');

    // Then - exits zero and emits the canonical JSON envelope
    expect(result.exitCode).toBe(0);
    expect(result.json).toMatch('valid.json');
});

test('emits a JSON report listing rule violations', async () => {
    // Given - check-invalid-tool references a nonexistent built-in
    await using result = await cli.fixture('$FIXTURES/check-invalid-tool/').exec('check --json');

    // Then - exits non-zero and the issue list is structurally stable
    expect(result.exitCode).toBe(1);
    expect(result.json).toMatch('invalid-tool.json');
});

test('--deep --json tags compile issues with source=compile', async () => {
    // Given - an overlay with an empty AGENTS.md, checked deep as JSON
    await using result = await cli
        .fixture('$FIXTURES/single-agent/')
        .fixture('empty-agents-md/')
        .exec('check --deep --json');

    // Then - the report flags a compile-sourced issue (scalpel: the compile issue set is dynamic)
    expect(result.exitCode).toBe(1);
    const report = result.json.value as {
        issues: { level: string; message: string; source?: string }[];
        summary: { errors: number };
        valid: boolean;
    };
    expect(report.valid).toBe(false);
    const compileIssues = report.issues.filter((issue) => issue.source === 'compile');
    expect(compileIssues.length).toBeGreaterThan(0);
    expect(required(compileIssues[0], 'the first compile issue').message).toContain('agent prompt');
});
