import type { PortsSharedPort } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { Openable } from '../openable';

const SHARE: PortsSharedPort = {
    bind: '0.0.0.0',
    detail: '',
    guest_port: 5173,
    http: true,
    last_seen_at: null,
    pinned: false,
    process: 'node',
    profile: '',
    reserved: 40_001,
    state: 'shared',
    urls: { local: 'http://127.0.0.1:40001', public: 'https://mac.tail1234.ts.net:40001' },
    vm: 'linux-1',
};

describe('openable', () => {
    test('an answer that gives no value makes nothing to open', () => {
        // Given - no command, a raw TCP share and a share with no address
        const answers = [
            Openable.ofSsh({ Address: 'api', Command: '', User: 'admin' }),
            Openable.ofShare({ ...SHARE, http: false }, true),
            Openable.ofShare({ ...SHARE, urls: { local: '', public: '' } }, true),
        ];

        // When - each is read
        const made = answers.filter((one) => one !== undefined);

        // Then - none can be opened
        expect(made).toStrictEqual([]);
    });

    test("a share's local address opens from its own Mac alone, its public one from any", () => {
        // Given - the same share with a public address, and with a local one alone
        const localOnly = { ...SHARE, urls: { ...SHARE.urls, public: '' } };

        // When - each is read on this Mac's row and on another Mac's
        const opens = [
            Openable.ofShare(SHARE, false)?.value,
            Openable.ofShare(localOnly, true)?.value,
            Openable.ofShare(localOnly, false)?.value,
        ];

        // Then - the public address anywhere, the local one on this Mac only
        expect(opens).toStrictEqual([
            'https://mac.tail1234.ts.net:40001',
            'http://127.0.0.1:40001',
            undefined,
        ]);
    });

    test("a release's notes open as the engine named them, and a release without them opens nothing", () => {
        // Given - a reading with notes, one with empty notes, one with none
        const head = { Sentence: 'Spwn v0.22.0 is out.', Tone: 'new' } as const;
        const notes = 'https://github.com/jterrazz/spwn/releases/tag/v0.22.0';
        const readings = [
            { Behind: true, Head: head, Installed: 'v0.21.0', Notes: notes },
            { Behind: true, Head: head, Installed: 'v0.21.0', Notes: '' },
            { Behind: true, Head: head, Installed: 'v0.21.0' },
        ];

        // When - each is read
        const opens = readings.map((release) => Openable.ofRelease(release)?.value);

        // Then - the url as sent, then nothing twice
        expect(opens).toStrictEqual([notes, undefined, undefined]);
    });
});
