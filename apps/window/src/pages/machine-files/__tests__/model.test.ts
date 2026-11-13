import { describe, expect, test } from 'vitest';

import { extendedBy, factsOf, filesOf } from '../model';

const origin = (Name: string, Layer: string) => ({ File: `/files/${Name}.yaml`, Layer, Name });

describe('model', () => {
    test('the files a machine file is read from are each listed once, in the engine order', () => {
        // Given - an explain that names a file twice and one file with no name
        const files = filesOf({
            Files: [
                origin('jterrazz-macbook-pro', 'user'),
                origin('station-macos', 'built-in'),
                origin('jterrazz-macbook-pro', 'user'),
                origin('', 'built-in'),
            ],
            Name: 'jterrazz-macbook-pro',
            Values: null,
        });

        // Then - each named file once
        expect(files.map((file) => file.Name)).toStrictEqual([
            'jterrazz-macbook-pro',
            'station-macos',
        ]);
    });

    test('an explain that names no file lists none', () => {
        // Given - an explain whose files are null on the wire
        const files = filesOf({ Files: null, Name: 'station-linux', Values: null });

        // Then - the list is empty
        expect(files).toStrictEqual([]);
    });

    test("a file's facts are its values, and it extends the files read before it", () => {
        // Given - an explain of a Mac's file that a mixin comes before
        const explain = {
            Files: [origin('jterrazz-base', 'user'), origin('jterrazz-macbook-pro', 'user')],
            Name: 'jterrazz-macbook-pro',
            Values: [
                {
                    Key: 'machine.kind',
                    Origin: origin('jterrazz-macbook-pro', 'user'),
                    Text: 'computer',
                },
                {
                    Key: 'machine.system.os',
                    Origin: origin('jterrazz-macbook-pro', 'user'),
                    Text: 'macos',
                },
                {
                    Key: 'machine.system.arch',
                    Origin: origin('jterrazz-base', 'user'),
                    Text: 'arm64',
                },
            ],
        };

        // When - the facts and the files extended are read for the file, then for its mixin
        const file = [
            factsOf(explain, 'jterrazz-macbook-pro'),
            extendedBy(explain, 'jterrazz-macbook-pro'),
        ];
        const mixin = [factsOf(explain, 'jterrazz-base'), extendedBy(explain, 'jterrazz-base')];

        // Then - the file makes a computer on macos and extends the mixin; the mixin sets the arch alone
        expect(file).toStrictEqual([
            { makes: 'computer', system: 'macos · arm64' },
            [{ identity: 'blue', name: 'jterrazz-base' }],
        ]);
        expect(mixin).toStrictEqual([{ makes: '', system: 'arm64' }, []]);
    });
});
