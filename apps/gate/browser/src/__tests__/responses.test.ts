import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { APP, gate, idOf } from './server.fixtures.ts';

const DATA = { items: [1, 2] };

async function onSite() {
    const sidecar = await gate();
    const { body } = await sidecar.call('POST', '/sessions', { provider: 'site' });
    const path = `/sessions/${idOf(body)}`;
    await sidecar.call('POST', `${path}/goto`, { url: sidecar.site });
    const fetched = async (address: string) =>
        await sidecar.call('POST', `${path}/eval`, {
            script: `fetch('${address}').then(() => 'sent')`,
        });
    return {
        fetched,
        path,
        sidecar,
        async [Symbol.asyncDispose]() {
            await sidecar[Symbol.asyncDispose]();
        },
    };
}

describe('responses', () => {
    test('awaits a response by its address, its method and a JSON type unless any is allowed', async () => {
        // Given - a session on the site
        await using site = await onSite();
        const { fetched, path, sidecar } = site;
        const wait = async (body: Record<string, unknown>, address: string) => {
            const [reply] = await Promise.all([
                sidecar.call('POST', `${path}/wait-response`, body),
                fetched(address),
            ]);
            return reply;
        };

        // When - a tool awaits JSON, a text it allows, a GET by a pattern with a letter class, and JSON past a text and a typeless body
        const replies = [
            await wait({ url_pattern: String.raw`data\.json$` }, '/data.json'),
            await wait({ allow_non_json: true, url_pattern: 'note' }, '/note.txt'),
            await wait({ method: 'get', url_pattern: String.raw`d\p{Ll}ta` }, '/data.json'),
        ];
        const [past] = await Promise.all([
            sidecar.call('POST', `${path}/wait-response`, { url_pattern: 'note|bare|data' }),
            fetched('/note.txt').then(async () => {
                await fetched('/bare');
                return await fetched('/data.json');
            }),
        ]);

        // Then - each response read whole, the text and the typeless body skipped where JSON alone was awaited
        const data = {
            body: { body: DATA, status: 200, url: `${sidecar.site}/data.json` },
            status: 200,
        };
        expect([...replies, past]).toStrictEqual([
            data,
            { body: { body: 'a note', status: 200, url: `${sidecar.site}/note.txt` }, status: 200 },
            data,
            data,
        ]);
    });

    test('a redirect answers no body, and a wait past its budget is refused', async () => {
        // Given - a session on the site
        await using site = await onSite();
        const { fetched, path, sidecar } = site;

        // When - a tool awaits the redirect a fetch follows, then a POST that never comes
        const [hop] = await Promise.all([
            sidecar.call('POST', `${path}/wait-response`, { url_pattern: 'hop' }),
            fetched('/hop'),
        ]);
        const [late] = await Promise.all([
            sidecar.call('POST', `${path}/wait-response`, {
                method: 'post',
                timeout_ms: 300,
                url_pattern: 'data',
            }),
            fetched('/data.json'),
        ]);

        // Then - the 302 with no body, and the wait's own timeout
        expect([hop, late]).toStrictEqual([
            { body: { body: null, status: 302, url: `${sidecar.site}/hop` }, status: 200 },
            {
                body: {
                    code: null,
                    error: 'page.waitForResponse: Timeout 300ms exceeded while waiting for event "response"',
                },
                status: 500,
            },
        ]);
    });

    test('the captured responses are the page’s JSON and scripts, filtered by address and time', async () => {
        // Given - at a known instant, a session that loaded the site's script, then fetched JSON, a redirect, a text and a typeless body
        using _ = clock.at('2026-10-09T08:00:00Z');
        await using site = await onSite();
        const { fetched, path, sidecar } = site;
        await fetched('/data.json');
        await fetched('/hop');
        await fetched('/note.txt');
        await fetched('/bare');
        const read = async (body: Record<string, unknown>) =>
            await sidecar.call('POST', `${path}/captured-responses`, body);

        // When - a tool reads them all, those since the instant they came, those of a pattern with a letter class, and those after
        const replies = [
            await read({}),
            await read({ since_ts: Date.parse('2026-10-09T08:00:00Z') }),
            await read({ url_pattern: String.raw`d\p{Ll}ta` }),
            await read({ since_ts: Date.parse('2026-10-09T08:00:00.001Z') }),
        ];

        // Then - the script, the JSON twice (fetched, then reached by the redirect) and the redirect with no body
        const at = Date.parse('2026-10-09T08:00:00Z');
        const entry = (url: string, body: unknown, status = 200) => ({
            body,
            method: 'GET',
            status,
            ts: at,
            url: `${sidecar.site}${url}`,
        });
        const data = entry('/data.json', DATA);
        const all = {
            body: { responses: [entry('/app.js', APP), data, entry('/hop', null, 302), data] },
            status: 200,
        };
        expect(replies).toStrictEqual([
            all,
            all,
            { body: { responses: [data, data] }, status: 200 },
            { body: { responses: [] }, status: 200 },
        ]);
    });
});
