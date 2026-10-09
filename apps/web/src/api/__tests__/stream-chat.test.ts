import { http, intercept } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { ActivityBlock } from '../../activity/blocks';
import { streamChat } from '../stream-chat';
import type { StreamMeta } from '../stream-chat';

const TALK = 'http://spwn.test/api/worlds/w-1/talk';
const END = { data: '[DONE]' };

function assistant(content: Record<string, unknown>[]) {
    return { data: { message: { content }, type: 'assistant' } };
}

async function talk() {
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
        onText: (text) => {
            heard.texts.push(text);
        },
        url: TALK,
    });
    return heard;
}

describe('streamChat', () => {
    test('an assistant reply streamed as events arrives as its blocks, then the result prices it', async () => {
        // Given - a text block, a tool call, then the result event that prices the turn
        await using _ = await intercept(
            http.post(TALK),
            http.sse([
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
                END,
            ]),
        );

        // Then - each block once, in order, and the turn's cost and duration
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
});
