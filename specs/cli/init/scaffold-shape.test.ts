import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * What the starter scaffold must NOT carry. `files:` says what a file
 * contains, never what it does not, so the scaffolding session stays in
 * `starter-scaffold.spec.yaml` — asserted whole by `cli.run()`, banners,
 * exit code and every file it wrote — and the three absence rules live
 * here, where a regex can express them.
 */

test('the scaffolded agent.yaml carries no colon-form local ref', async () => {
    // Given - the scaffolding session, stated in the document
    const result = await cli.run('starter-scaffold.spec.yaml');

    // Then - local refs are path-form only; the retired `skill:`/`tool:`/`hook:` forms are gone
    const agentYaml = result.file('spwn/agents/neo/agent.yaml').content;
    expect(agentYaml).not.toMatch(/\bskill:[a-z]/);
    expect(agentYaml).not.toMatch(/\btool:[a-z]/);
    expect(agentYaml).not.toMatch(/\bhook:[a-z]/);
});

test('the manifest points knowledge at spwn/knowledge, never at the root', async () => {
    // Given - the same session, pinning the 2026-04 knowledge relocation
    const result = await cli.run('starter-scaffold.spec.yaml');

    // Then - the retired root-level path never sneaks back in
    expect(result.file('spwn.yaml').content).not.toMatch(/knowledge: \.\/knowledge$/m);
});

test('the default scaffold leaves the runtime backend unpinned', async () => {
    // Given - the same session; the resolver should pick the backend at spawn time
    const result = await cli.run('starter-scaffold.spec.yaml');

    // Then - no non-comment line declares a runtime block or a backend key
    for (const raw of result.file('spwn/agents/neo/agent.yaml').content.split('\n')) {
        const line = raw.trimStart();
        if (line.startsWith('#')) {
            continue;
        }
        expect(line).not.toMatch(/^runtime:\s*$/);
        expect(line).not.toMatch(/^backend:/);
    }
});
