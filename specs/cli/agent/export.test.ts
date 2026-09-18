import { execSync } from 'node:child_process';
import { join } from 'node:path';
import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * The one export spec whose subject is INSIDE the archive. The framework
 * reads file text, not tar entries, so this shells out to `tar tzf` over
 * the cwd — genuine test plumbing, not a file assertion. The session
 * itself stays in `exported-archive.spec.yaml`, asserted whole by
 * `cli.run()`; the code adds the listing.
 *
 * Every other export and import spec is a document beside this file.
 */

test('the exported archive carries every mind layer', async () => {
    // Given - the create + export session, stated in the document
    const result = await cli.run('exported-archive.spec.yaml');

    // Then - the tarball listing carries the Soul plus the two Mind layer dirs
    const listing = execSync(`tar tzf ${join(result.filesystem.cwd, 'neo.tar.gz')}`, {
        encoding: 'utf8',
    });
    expect(listing).toContain('SOUL.md');
    expect(listing).toMatch(/(?:^|\n)playbooks(?:\/|\n|$)/);
    expect(listing).toMatch(/(?:^|\n)journal(?:\/|\n|$)/);
});
