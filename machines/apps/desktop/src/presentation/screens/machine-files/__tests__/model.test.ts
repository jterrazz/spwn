import type { MachinefileFileView, MachinefileHead } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { extendedBy, linesOf, makesOf, shelvedOf, sizeOf, systemOf } from '../model';

const view = (
    Name: string,
    Shelf: MachinefileFileView['Shelf'],
    Extends: null | string[] = null,
): MachinefileFileView => ({
    Apps: null,
    Description: '',
    Extends,
    Followers: null,
    Name,
    Origin: 'user',
    Path: '',
    Settings: null,
    Shelf,
    Text: '',
    Verdict: { Sentence: 'No machine follows it.', Tone: 'plain' },
});

describe('model', () => {
    test('the files are shelved yours, mixins, then shipped, a shelf with no file left out', () => {
        // Given - files in the engine's name order, none of them a mixin but one
        const files = [
            view('base', 'mixin'),
            view('macbook-pro', 'yours'),
            view('base-macos', 'shipped'),
            view('base-ubuntu', 'shipped'),
        ];

        // When - they are shelved
        const shelves = shelvedOf(files).map(({ shelf, files: held }) => [
            shelf,
            held.map((file) => file.Name),
        ]);

        // Then - the order of the captions, each shelf keeping the engine's order
        expect(shelves).toStrictEqual([
            ['yours', ['macbook-pro']],
            ['mixin', ['base']],
            ['shipped', ['base-macos', 'base-ubuntu']],
        ]);
        expect(shelvedOf([view('only', 'yours')])).toHaveLength(1);
    });

    test('the files that extend a file are the ones whose Extends name it', () => {
        // Given - a mixin extended by two files, and a file that extends nothing
        const files = [
            view('base', 'mixin'),
            view('a', 'yours', ['base']),
            view('b', 'yours', ['base', 'other']),
            view('c', 'yours'),
        ];

        // Then - the join inverts Extends
        expect(extendedBy(files, 'base').map((file) => file.Name)).toStrictEqual(['a', 'b']);
        expect(extendedBy(files, 'a')).toStrictEqual([]);
    });

    test("a file's head is worded by the engine's tables, its VM's size in counts of the wire", () => {
        // Given - a head that makes a workspace on Ubuntu, and one that makes a Mac
        const workspace: MachinefileHead = {
            Kind: 'workspace',
            System: { Arch: 'arm64', OS: 'ubuntu', Version: '24.04' },
            VM: {
                CPUs: 4,
                Disk: '20G',
                From: 'base-ubuntu',
                Lifetime: 'ephemeral',
                Memory: '4G',
            },
        };
        const mac: MachinefileHead = {
            Kind: 'computer',
            System: { Arch: 'arm64', OS: 'macos' },
        };
        const silent: MachinefileHead = { Kind: 'computer', System: {} };

        // Then - the role and the system, the size apart, a Mac having none
        expect([makesOf(workspace), systemOf(workspace), sizeOf(workspace)]).toStrictEqual([
            'a workspace',
            'Ubuntu 24.04 · arm64',
            '4 CPU · 4G · 20G',
        ]);
        expect([makesOf(mac), systemOf(mac), sizeOf(mac)]).toStrictEqual([
            'a Mac',
            'macOS · arm64',
            '',
        ]);
        expect(systemOf(silent)).toBe('');
    });

    test('a file text is numbered line by line, its last newline ending no line', () => {
        // Given - a text of two lines
        // Then - two numbered lines, and an empty text has none
        expect(linesOf('name: a\napps: []\n')).toStrictEqual([
            { number: 1, text: 'name: a' },
            { number: 2, text: 'apps: []' },
        ]);
        expect(linesOf('')).toStrictEqual([]);
    });
});
