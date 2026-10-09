import { file } from '@jterrazz/test';
import { execFileSync } from 'node:child_process';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read inside an archive: the exported tarball is listed with tar

describe('export', () => {
    test('the exported archive carries the soul and both mind layers', async () => {
        // Given - the create and export session, stated whole in the document
        const result = await cli.run(file('./exported-archive.spec.yaml'));

        // Then - the archive lists the soul and the two layer folders, read with tar since the result reads files, not archive entries
        const listing = execFileSync('tar', ['tzf', join(result.filesystem.cwd, 'neo.tar.gz')], {
            encoding: 'utf8',
        });
        expect(
            listing
                .split('\n')
                .filter((entry) => entry !== '')
                .toSorted(),
        ).toStrictEqual(['.', 'AGENTS.md', 'SOUL.md', 'agent.yaml', 'journal', 'playbooks']);
    });
});
