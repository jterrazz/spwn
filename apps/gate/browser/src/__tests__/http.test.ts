import { once } from 'node:events';
import { createServer } from 'node:http';
import type { IncomingMessage, ServerResponse } from 'node:http';
import { describe, expect, test } from 'vitest';

import { readJson, RefusedError, send } from '../http.ts';

async function served(handle: Parameters<typeof createServer>[1]) {
    const server = createServer(handle);
    server.listen(0, '127.0.0.1');
    await once(server, 'listening');
    const address = server.address();
    const origin = `http://127.0.0.1:${typeof address === 'object' && address !== null ? address.port : 0}`;
    return {
        origin,
        [Symbol.dispose]() {
            server.close();
        },
    };
}

async function echoed(request: IncomingMessage, response: ServerResponse): Promise<void> {
    try {
        send(response, 200, await readJson(request));
    } catch (error) {
        const refused = error instanceof RefusedError ? error : null;
        send(response, refused?.status ?? 500, { message: refused?.message ?? String(error) });
    }
}

describe('http', () => {
    test('a refusal carries its status, its message and the code a tool branches on, none by default', () => {
        // Given - a refusal with a code, and one without
        const coded = new RefusedError(409, 'session busy', 'busy');
        const plain = new RefusedError(400, 'bad request');

        // Then - each reads whole
        expect([coded, plain].map((e) => [e.name, e.status, e.message, e.code])).toStrictEqual([
            ['RefusedError', 409, 'session busy', 'busy'],
            ['RefusedError', 400, 'bad request', null],
        ]);
    });

    test('send answers a string as plain text and any other value as JSON, each with its byte length', async () => {
        // Given - a server answering one text and one object
        using site = await served((request, response) => {
            if (request.url === '/text') {
                send(response, 200, 'héllo');
                return;
            }
            send(response, 201, { word: 'héllo' });
        });

        // When - both are fetched
        const read = async (path: string) => {
            const reply = await fetch(`${site.origin}${path}`);
            return {
                body: await reply.text(),
                length: reply.headers.get('content-length'),
                status: reply.status,
                type: reply.headers.get('content-type'),
            };
        };
        const answers = { json: await read('/json'), text: await read('/text') };

        // Then - the text is not quoted, the object is serialized, and each length counts bytes
        expect(answers).toStrictEqual({
            json: {
                body: '{"word":"héllo"}',
                length: '17',
                status: 201,
                type: 'application/json',
            },
            text: { body: 'héllo', length: '6', status: 200, type: 'text/plain' },
        });
    });

    test('readJson reads an object, an empty body as {}, anything else as {}, and a broken body as a refusal', async () => {
        // Given - a server echoing what readJson makes of each body
        using site = await served((request, response) => {
            void echoed(request, response);
        });

        // When - several bodies are posted
        const post = async (body: string) => {
            const reply = await fetch(site.origin, { body, method: 'POST' });
            return [reply.status, (await reply.json()) as unknown];
        };
        const read = {
            broken: await post('{"a":'),
            empty: await post(''),
            null: await post('null'),
            number: await post('5'),
            object: await post('{"a":1,"b":{"c":2}}'),
            text: await post('"abc"'),
        };

        // Then - only an object comes through, and the parser's own words name the break
        expect(read).toStrictEqual({
            broken: [400, { message: expect.stringMatching(/^bad json: /u) as unknown }],
            empty: [200, {}],
            null: [200, {}],
            number: [200, {}],
            object: [200, { a: 1, b: { c: 2 } }],
            text: [200, {}],
        });
    });
});
