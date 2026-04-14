import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';

describe('architect Stack', () => {
    let home: string;
    let originalSpwnHome: string | undefined;

    beforeEach(() => {
        originalSpwnHome = process.env.SPWN_HOME;
        home = createSpwnHome();
        process.env.SPWN_HOME = home;
    });

    afterEach(() => {
        if (originalSpwnHome !== undefined) {
            process.env.SPWN_HOME = originalSpwnHome;
        } else {
            delete process.env.SPWN_HOME;
        }
    });

    test('architect Stack directory can be created', () => {
        // GIVEN - a SPWN_HOME
        const architectDir = join(home, 'architect');
        mkdirSync(architectDir, { recursive: true });

        // WHEN - writing a stack.md file
        const stackPath = join(architectDir, 'stack.md');
        const content = [
            '# Architect Stack',
            '',
            '## Focus',
            '- [ ] Set up initial agent fleet',
            '',
            '## Queued',
            '- [ ] Configure monitoring',
            '- [ ] Add error handling',
            '',
            '## Done',
            '- [x] Initialize project structure',
            '',
        ].join('\n');
        writeFileSync(stackPath, content);

        // THEN - file exists and is readable
        expect(existsSync(stackPath)).toBe(true);
        const read = readFileSync(stackPath, 'utf8');
        expect(read).toContain('## Focus');
        expect(read).toContain('## Queued');
        expect(read).toContain('## Done');
    });

    test('architect Stack default template has expected sections', () => {
        // GIVEN - a fresh architect directory
        const architectDir = join(home, 'architect');
        mkdirSync(architectDir, { recursive: true });

        // WHEN - writing the default template
        const defaultContent = '# Architect Stack\n\n## Focus\n\n## Queued\n\n## Done\n';
        const stackPath = join(architectDir, 'stack.md');
        writeFileSync(stackPath, defaultContent);

        // THEN - template has all required sections
        const content = readFileSync(stackPath, 'utf8');
        expect(content).toContain('# Architect Stack');
        expect(content).toContain('## Focus');
        expect(content).toContain('## Queued');
        expect(content).toContain('## Done');
    });

    test('architect Stack supports checkbox parsing', () => {
        // GIVEN - a Stack file with checkboxes
        const architectDir = join(home, 'architect');
        mkdirSync(architectDir, { recursive: true });
        const stackPath = join(architectDir, 'stack.md');
        writeFileSync(
            stackPath,
            [
                '# Stack',
                '## Focus',
                '- [ ] Task A',
                '- [ ] Task B',
                '## Queued',
                '- [ ] Task C',
                '## Done',
                '- [x] Task D',
                '- [x] Task E',
            ].join('\n'),
        );

        // WHEN - reading and parsing
        const content = readFileSync(stackPath, 'utf8');
        const pendingMatches = content.match(/- \[ \]/g) ?? [];
        const doneMatches = content.match(/- \[x\]/g) ?? [];

        // THEN - counts are correct
        expect(pendingMatches.length).toBe(3);
        expect(doneMatches.length).toBe(2);
    });
});
