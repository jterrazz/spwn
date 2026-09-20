import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * The two dependency-grammar specs whose subject is what is ABSENT from the
 * manifest: no entry may survive without an explicit scheme. `files:` says what
 * a file contains, never what it does not, so the session stays in its document
 * — `promoted-bare-ref.spec.yaml` and `explicit-manifest-schemes.spec.yaml`,
 * both asserted in full by `cli.run()` — and the code adds the one probe the
 * format has no vocabulary for.
 */

/** Every list entry that carries a bare name, i.e. no `<scheme>:` and no `<kind>/`. */
function bareEntries(manifest: string): null | string[] {
    return manifest.match(/^\s*-\s+[a-z0-9][a-z0-9-]*\s*$/gm);
}

test('a bare name accepted on the cli never lands bare in agent.yaml', async () => {
    // Given - the whole session, stated in the document: init, install python, check
    const result = await cli.run('promoted-bare-ref.spec.yaml');

    // Then - the promotion left nothing bare behind
    const manifest = result.file('spwn/agents/neo/agent.yaml').content;
    expect(manifest).not.toMatch(/^\s*-\s+python\s*$/m);
});

test('no manifest entry survives without an explicit scheme', async () => {
    // Given - init followed by three installs mixing bare, local and catalog refs
    const result = await cli.run('explicit-manifest-schemes.spec.yaml');

    // Then - every entry on disk carries its scheme
    const bare = bareEntries(result.file('spwn/agents/neo/agent.yaml').content);
    expect(bare, `manifest carries bare entries: ${bare?.join(', ')}`).toBeNull();
});
