import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

import { RefusedError } from './http.ts';

type Cookie = {
    domain: string;
    httpOnly: boolean;
    name: string;
    path: string;
    sameSite: 'Lax';
    secure: boolean;
    value: string;
};

function domainsOf(credentials: string, provider: string): string[] {
    const hint = join(credentials, provider, '.domains');
    // outside: the gate writes the provider's hosts here when it registers it; the fallback guesses the provider's own .com
    return existsSync(hint)
        ? readFileSync(hint, 'utf8')
              .split('\n')
              .map((line) => line.trim())
              .filter((line) => line !== '')
        : [`${provider}.com`];
}

/** The cookies the extension synced for `provider`, laid on every host the provider is known by. */
export function providerCookies(credentials: string, provider: string): Cookie[] {
    const file = join(credentials, provider, 'cookies.json');
    if (!existsSync(file)) {
        throw new RefusedError(
            500,
            `no cookies for provider "${provider}" — run the cookie-sync extension`,
            'NO_COOKIES',
        );
    }
    const synced: unknown = JSON.parse(readFileSync(file, 'utf8'));
    const jar: unknown =
        typeof synced === 'object' && synced !== null ? Reflect.get(synced, 'cookies') : undefined;
    const entries = typeof jar === 'object' && jar !== null ? Object.entries(jar) : [];
    return domainsOf(credentials, provider).flatMap((host) =>
        entries.map(([name, value]) => ({
            domain: host.startsWith('.') ? host : `.${host}`,
            httpOnly: false,
            name,
            path: '/',
            sameSite: 'Lax' as const,
            secure: true,
            value: String(value),
        })),
    );
}
