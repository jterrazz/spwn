import { describe, expect, test } from 'vitest';

import { looksSecret, screen, shownOrWithheld, WITHHELD } from '../secret';

const GITHUB_TOKEN = ['ghp', '0123456789abcdefghijABCDEFGHIJ'].join('_');

describe('secret', () => {
    test('a value shaped like a token, a key or a long random string is withheld', () => {
        // Given - one value of each shape a credential takes
        const values = [
            GITHUB_TOKEN,
            ['gho', '16C7e42F292c6912E7710c838347Ae178B4a'].join('_'),
            ['sk', 'ant-oat01-abcdefghijklmnop'].join('-'),
            ['npm', 'abcdefghijklmnopqrstuvwxyz0123'].join('_'),
            ['eyJhbGciOiJIUzI1NiJ9', 'eyJzdWIiOiIxMjM0In0', 'c2lnbmF0dXJl'].join('.'),
            ['-----BEGIN OPENSSH', 'PRIVATE KEY-----'].join(' '),
            'token: 4f1c9a0b7e2d4c6a8b1e3f5a7c9d0e2b4f6a8c1e',
            `secret: ${['abcdEFGH1234', 'ijklMNOP5678', 'qrstUVWX9012abcd'].join('/')}`,
        ];

        // Then - every one is withheld
        expect(values.filter((value) => !looksSecret(value))).toStrictEqual([]);
    });

    test('a name, a scope, a path or a command is shown', () => {
        // Given - what an account row carries when nothing leaked
        const values = [
            'jterrazz',
            'repo, workflow',
            '~/.claude/.credentials.json',
            'keychain: operator:claude:oauth-token',
            'station accounts share claude --with api',
            'jterrazz-macbook-pro.tail1234.ts.net',
        ];

        // Then - none is withheld
        expect(values.filter((value) => looksSecret(value))).toStrictEqual([]);
    });

    test('a long path and a digest are shown, though each is a long run of letters and digits', () => {
        // Given - an absolute path, a path in the home folder and an image digest, each past 32 characters
        const digest = [
            '9f86d081884c7d659a2feaa0c55ad015',
            'a3bf4f1b2b0b822cd15d6c15b0f00a08',
        ].join('');
        const values = [
            '/Users/jterrazz/Developer/studio-os/home/studio/work/worktrees/window-l2/apps',
            '~/Library/Application-Support/com-jterrazz-station-2026/accounts-v2/github',
            `ghcr.io/jterrazz/station@sha256:${digest}`,
            `sha512:${digest}${digest}`,
        ];

        // Then - none is withheld
        expect(values.filter((value) => looksSecret(value))).toStrictEqual([]);
    });

    test('a drawn value or error text is Withheld when it holds a secret, and shown as given otherwise', () => {
        // Given - an error carrying a token, a machine-file value carrying one, and three values that must still show
        const digest = [
            '9f86d081884c7d659a2feaa0c55ad015',
            'a3bf4f1b2b0b822cd15d6c15b0f00a08',
        ].join('');
        const values = [
            `git clone refused: remote said bad credentials for ${GITHUB_TOKEN}`,
            ['npm', 'abcdefghijklmnopqrstuvwxyz0123'].join('_'),
            '/opt/homebrew/Cellar/node/25.1.0/bin/node',
            '2026.10.20',
            `ghcr.io/jterrazz/station-base@sha256:${digest}`,
        ];

        // Then - the two secrets read Withheld, the path, the version and the digest read as given
        expect(values.map(shownOrWithheld)).toStrictEqual([
            WITHHELD,
            WITHHELD,
            '/opt/homebrew/Cellar/node/25.1.0/bin/node',
            '2026.10.20',
            `ghcr.io/jterrazz/station-base@sha256:${digest}`,
        ]);
    });

    test('a field that holds a secret is named, never its value', () => {
        // Given - an account whose detail carries a token
        const result = screen([
            ['identity', 'jterrazz'],
            ['detail', GITHUB_TOKEN],
            ['hint', undefined],
        ]);

        // Then - the identity is shown, the detail is reported by its name alone
        expect(result).toStrictEqual({ shown: ['jterrazz'], withheld: ['detail'] });
    });
});
