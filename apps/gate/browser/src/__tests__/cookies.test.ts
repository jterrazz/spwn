import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { providerCookies } from '../cookies.ts';
import { RefusedError } from '../http.ts';

function credentials(files: Record<string, string>) {
    const folder = mkdtempSync(join(tmpdir(), 'gate-cookies-'));
    mkdirSync(join(folder, 'x'));
    for (const [name, content] of Object.entries(files)) {
        writeFileSync(join(folder, 'x', name), content);
    }
    return {
        folder,
        [Symbol.dispose]() {
            rmSync(folder, { force: true, recursive: true });
        },
    };
}

const cookie = (domain: string, name: string, value: string) => ({
    domain,
    httpOnly: false,
    name,
    path: '/',
    sameSite: 'Lax',
    secure: true,
    value,
});

describe('providerCookies', () => {
    test('lays every synced cookie on every host the provider names, each as a domain cookie', () => {
        // Given - two cookies, one a number, and two hosts, one already a domain, between blank lines
        using jar = credentials({
            '.domains': '\n  x.com  \n.twitter.com\n\n',
            'cookies.json': JSON.stringify({ cookies: { auth_token: 'secret', ct0: 42 } }),
        });

        // When - the provider's cookies are read
        const cookies = providerCookies(jar.folder, 'x');

        // Then - each cookie on each host, its value a string
        expect(cookies).toStrictEqual([
            cookie('.x.com', 'auth_token', 'secret'),
            cookie('.x.com', 'ct0', '42'),
            cookie('.twitter.com', 'auth_token', 'secret'),
            cookie('.twitter.com', 'ct0', '42'),
        ]);
    });

    test('without the gate’s hosts, guesses the provider’s own .com', () => {
        // Given - one cookie and no hosts file
        using jar = credentials({ 'cookies.json': JSON.stringify({ cookies: { session: 'a' } }) });

        // When - the provider's cookies are read
        const cookies = providerCookies(jar.folder, 'x');

        // Then - laid on .x.com
        expect(cookies).toStrictEqual([cookie('.x.com', 'session', 'a')]);
    });

    test('a sync holding no cookie object lays none', () => {
        // Given - five syncs: no cookies key, cookies that are a string or null, a sync that is null or a number
        const read = ['{}', '{"cookies":"none"}', '{"cookies":null}', 'null', '5'].map((synced) => {
            using jar = credentials({ 'cookies.json': synced });
            return providerCookies(jar.folder, 'x');
        });

        // Then - nothing for any of them
        expect(read).toStrictEqual([[], [], [], [], []]);
    });

    test('a provider never synced is refused with its code', () => {
        // Given - a provider folder with no cookies.json
        using jar = credentials({});

        // When - its cookies are read
        let refused: unknown;
        try {
            providerCookies(jar.folder, 'x');
        } catch (error) {
            refused = error;
        }

        // Then - a 500 the tool can branch on
        expect(refused).toStrictEqual(
            new RefusedError(
                500,
                'no cookies for provider "x": run the cookie-sync extension',
                'NO_COOKIES',
            ),
        );
        expect(
            refused instanceof RefusedError ? [refused.status, refused.code] : null,
        ).toStrictEqual([500, 'NO_COOKIES']);
    });
});
