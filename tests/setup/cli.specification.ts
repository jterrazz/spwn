import { cli } from '@jterrazz/test';
import { resolve } from 'node:path';

/**
 * CLI specification runner for spwn.
 *
 * Each spec call gets a fresh temporary directory. `.project('name')`
 * copies `tests/fixtures/<name>/` into that directory before exec.
 * The spwn binary is run with that directory as its working directory,
 * so any "find a project" walk discovers the fixture as the project
 * root.
 *
 * This runner is the new shape — see tests/setup/spwn.specification.ts
 * for the legacy `createTestContext` / `spwn()` helper that older tests
 * still use.
 */
const SPWN_BIN = resolve(import.meta.dirname, '../../bin/spwn');

export const spec = await cli({
    command: SPWN_BIN,
    root: '../fixtures',
});
