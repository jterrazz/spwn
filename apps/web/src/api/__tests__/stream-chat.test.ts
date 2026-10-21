import { http, intercept } from '@jterrazz/test/stubs';
import { describe, expect, test } from 'vitest';

import type { ActivityBlock } from '../../activity/blocks';
import { streamChat } from '../stream-chat';
import type { StreamMeta } from '../stream-chat';

const TALK = 'http://spwn.test/api/worlds/w-1/talk';
const END = { data: '[DONE]' };

function assistant(content: Record<string, unknown>[]) {
    return { data: { message: { content }, type: 'assistant' } };
}

async function talk(options: { signal?: AbortSignal; withText?: boolean } = {}) {
    const heard = {
        blocks: [] as ActivityBlock[][],
        done: [] as StreamMeta[],
        errors: [] as string[],
        texts: [] as string[],
    };
    await streamChat({
        body: { message: 'hi' },
        onBlocks: (blocks) => {
            heard.blocks.push(blocks);
        },
        onDone: (meta) => {
            heard.done.push(meta);
        },
        onError: (error) => {
            heard.errors.push(error);
        },
        ...(options.withText === false
            ? {}
            : {
                  onText: (text: string) => {
                      heard.texts.push(text);
                  },
              }),
        ...(options.signal === undefined ? {} : { signal: options.signal }),
        url: TALK,
    });
    return heard;
}

describe('streamChat', () => {
    test('an assistant reply streamed as events arrives as its blocks, then the result prices it', async () => {
        // Given - a text block sent twice, a tool call, the result event that prices the turn, and an event after it
        await using _ = await intercept(
            http.post(TALK),
            http.sse([
                assistant([{ text: 'hello world', type: 'text' }]),
                assistant([{ text: 'hello world', type: 'text' }]),
                assistant([{ id: 't1', input: { command: 'ls' }, name: 'bash', type: 'tool_use' }]),
                {
                    data: {
                        duration_ms: 1234,
                        subtype: 'success',
                        total_cost_usd: 0.05,
                        type: 'result',
                    },
                },
                { data: { subtype: 'end', type: 'system' } },
                END,
            ]),
        );

        // Then - each block once, in order, and the cost and duration of the result alone
        await expect(talk()).resolves.toStrictEqual({
            blocks: [
                [{ content: 'hello world', type: 'text' }],
                [{ id: 't1', input: { command: 'ls' }, tool: 'bash', type: 'tool_use' }],
            ],
            done: [{ cost: 0.05, duration: 1234 }],
            errors: [],
            texts: [],
        });
    });

    test('a thought and a failed result arrive as their own blocks', async () => {
        // Given - a thought, then a result event carrying a failure
        await using _ = await intercept(
            http.post(TALK),
            http.sse([
                assistant([{ text: 'Let me think', type: 'thinking' }]),
                { data: { result: 'Rate limit exceeded', subtype: 'error', type: 'result' } },
                END,
            ]),
        );

        // Then - the thought, then the error
        await expect(talk()).resolves.toStrictEqual({
            blocks: [
                [{ content: 'Let me think', type: 'thinking' }],
                [{ content: 'Rate limit exceeded', type: 'error' }],
            ],
            done: [{ cost: undefined, duration: undefined }],
            errors: [],
            texts: [],
        });
    });

    test('lines that are not events reach the text callback whole', async () => {
        // Given - a plain-text body, arriving in two pieces
        await using _ = await intercept(
            http.post(TALK),
            http.stream(['Hello plain response.\n', 'Second line.\n'], {
                contentType: 'text/plain',
            }),
        );

        // Then - each line, its line break kept
        await expect(talk()).resolves.toStrictEqual({
            blocks: [],
            done: [{}],
            errors: [],
            texts: ['Hello plain response.\n', 'Second line.\n'],
        });
    });

    test('a reply in one JSON body arrives as one text block', async () => {
        // Given - a backend that answered in one serialised body
        await using _ = await intercept(http.post(TALK), http.json({ response: 'one body' }));

        // Then - the reply's text, as a block
        await expect(talk()).resolves.toStrictEqual({
            blocks: [[{ content: 'one body', type: 'text' }]],
            done: [{}],
            errors: [],
            texts: [],
        });
    });

    test('a refusal reports the error the backend gave', async () => {
        // Given - a backend that said no, and said why
        await using _ = await intercept(
            http.post(TALK),
            http.error(500, { error: 'Internal server error' }),
        );

        // Then - the reason, and no completion
        await expect(talk()).resolves.toStrictEqual({
            blocks: [],
            done: [],
            errors: ['Internal server error'],
            texts: [],
        });
    });

    test('an unreachable backend reports the failed fetch', async () => {
        // Given - nothing answering
        await using _ = await intercept(http.post(TALK), http.unreachable());

        // When - the message is sent
        const heard = await talk();

        // Then - one error naming the failed fetch, and no completion
        expect({
            done: heard.done,
            errors: heard.errors.map((error) => /fetch failed|Failed to fetch/iu.test(error)),
        }).toStrictEqual({
            done: [],
            errors: [true],
        });
    });

    test('a plain stream keeps its lines whole across chunks, drops comments, and reads JSON that is no event as text', async () => {
        // Given - a CRLF line, a line cut across two chunks, an SSE comment, and three JSON lines that are no event
        await using _ = await intercept(
            http.post(TALK),
            http.stream(['Hello\r\nSpl', 'it line\n: ping\n', 'null\n{"type":3}\n{"kind":"x"}\n'], {
                contentType: 'text/plain',
            }),
        );

        // Then - every text line once, its break restored
        await expect(talk()).resolves.toStrictEqual({
            blocks: [],
            done: [{}],
            errors: [],
            texts: ['Hello\n', 'Split line\n', 'null\n', '{"type":3}\n', '{"kind":"x"}\n'],
        });
    });

    test('without a text callback, plain lines arrive as text blocks', async () => {
        // Given - a plain line and a listener of blocks alone
        await using _ = await intercept(
            http.post(TALK),
            http.stream(['Plain\n'], { contentType: 'text/plain' }),
        );

        // Then - the line as a block
        await expect(talk({ withText: false })).resolves.toStrictEqual({
            blocks: [[{ content: 'Plain\n', type: 'text' }]],
            done: [{}],
            errors: [],
            texts: [],
        });
    });

    test('a JSON body that says nothing reads as no response', async () => {
        // Given - a reply whose fields are all empty
        await using _ = await intercept(http.post(TALK), http.json({ output: '', response: '' }));

        // Then - the placeholder
        await expect(talk()).resolves.toMatchObject({
            blocks: [[{ content: 'No response', type: 'text' }]],
        });
    });

    test('a JSON body is read from its first field that holds text, and the request is JSON', async () => {
        // Given - a backend echoing the request's type and body in `message`, its `response` empty
        await using _ = await intercept(http.post(TALK), (request) =>
            http.json({
                message: `${request.headers['content-type']} ${JSON.stringify(request.body)}`,
                response: '',
            }),
        );

        // Then - the echo
        await expect(talk()).resolves.toMatchObject({
            blocks: [[{ content: 'application/json {"message":"hi"}', type: 'text' }]],
        });
    });

    test('an empty answer arrives as an empty text block', async () => {
        // Given - a backend that answered with no body
        await using _ = await intercept(http.post(TALK), http.empty(200));

        // Then - one empty reply
        await expect(talk()).resolves.toStrictEqual({
            blocks: [[{ content: '', type: 'text' }]],
            done: [{}],
            errors: [],
            texts: [],
        });
    });

    test('an answer with no body stream arrives as an empty text block', async () => {
        // Given - a 204, which carries no body stream at all
        await using _ = await intercept(http.post(TALK), http.empty(204));

        // Then - one empty reply
        await expect(talk()).resolves.toStrictEqual({
            blocks: [[{ content: '', type: 'text' }]],
            done: [{}],
            errors: [],
            texts: [],
        });
    });

    test('a refusal without a reason reports its status', async () => {
        // Given - a gateway error whose body names no error
        await using _ = await intercept(http.post(TALK), http.error(502, { message: 'upstream' }));

        // Then - the status
        await expect(talk()).resolves.toStrictEqual({
            blocks: [],
            done: [],
            errors: ['HTTP 502'],
            texts: [],
        });
    });

    test('a request cancelled by its caller reports the cancellation', async () => {
        // Given - an answering backend, and a signal aborted before the message is sent
        await using _ = await intercept(http.post(TALK), http.json({ response: 'too late' }));
        const controller = new AbortController();
        controller.abort();

        // Then - the cancellation, not the answer
        await expect(talk({ signal: controller.signal })).resolves.toStrictEqual({
            blocks: [],
            done: [],
            errors: ['Request cancelled'],
            texts: [],
        });
    });
});
