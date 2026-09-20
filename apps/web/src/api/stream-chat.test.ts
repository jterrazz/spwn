import { defineContracts, http, intercept } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { streamChat } from './stream-chat';

const primaryUrl = 'http://spwn.test/api/worlds/w-1/talk';
const fallbackUrl = 'http://fallback.test/api/architect/talk';

/** The frame that closes a Claude Code stream. */
const done = { data: '[DONE]' };

function callbacks() {
    const blocks: unknown[][] = [];
    const texts: string[] = [];
    const errors: string[] = [];
    const dones: { cost?: number | undefined; duration?: number | undefined }[] = [];

    return {
        blocks,
        texts,
        errors,
        dones,
        onBlocks: (value: unknown[]) => blocks.push(value),
        onText: (value: string) => texts.push(value),
        onDone: (value: { cost?: number | undefined; duration?: number | undefined }) =>
            dones.push(value),
        onError: (value: string) => errors.push(value),
    };
}

describe('streamChat', () => {
    test('parses assistant text from an SSE response', async () => {
        // Given - one assistant event, then the closing frame
        await using _ = await intercept(
            http.post(primaryUrl),
            http.sse([
                {
                    data: {
                        message: { content: [{ text: 'hello world', type: 'text' }] },
                        type: 'assistant',
                    },
                },
                done,
            ]),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.dones).toHaveLength(1);
        expect(cb.blocks.flat()).toContainEqual({ type: 'text', content: 'hello world' });
    });

    test('parses tool_use blocks from an SSE response', async () => {
        // Given - an assistant event whose only content is a tool call
        await using _ = await intercept(
            http.post(primaryUrl),
            http.sse([
                {
                    data: {
                        message: {
                            content: [
                                {
                                    id: 't1',
                                    input: { command: 'ls' },
                                    name: 'bash',
                                    type: 'tool_use',
                                },
                            ],
                        },
                        type: 'assistant',
                    },
                },
                done,
            ]),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'list files' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.blocks.flat()).toContainEqual({
            id: 't1',
            input: { command: 'ls' },
            tool: 'bash',
            type: 'tool_use',
        });
    });

    test('extracts cost and duration metadata from result events', async () => {
        // Given - a reply followed by the result event that prices it
        await using _ = await intercept(
            http.post(primaryUrl),
            http.sse([
                {
                    data: {
                        message: { content: [{ text: 'done', type: 'text' }] },
                        type: 'assistant',
                    },
                },
                {
                    data: {
                        duration_ms: 1234,
                        subtype: 'success',
                        total_cost_usd: 0.05,
                        type: 'result',
                    },
                },
                done,
            ]),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.dones).toStrictEqual([{ cost: 0.05, duration: 1234 }]);
    });

    test('handles plain text streams', async () => {
        // Given - a body that is not SSE at all, arriving in two pieces
        await using _ = await intercept(
            http.post(primaryUrl),
            http.stream(['Hello plain response.\n', 'Second line.\n'], {
                contentType: 'text/plain',
            }),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.dones).toHaveLength(1);
        expect(cb.texts).toContain('Hello plain response.\n');
    });

    test('handles JSON responses', async () => {
        // Given - a backend that answered in one serialised body
        await using _ = await intercept(
            http.post(primaryUrl),
            http.json({ response: 'JSON fallback response' }),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.dones).toHaveLength(1);
        expect(cb.blocks.flat()).toContainEqual({
            type: 'text',
            content: 'JSON fallback response',
        });
    });

    test('reports network errors from the primary URL', async () => {
        // Given - nothing answering at the primary URL
        await using _ = await intercept(http.post(primaryUrl), http.unreachable());

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.dones).toHaveLength(0);
        expect(cb.errors[0]).toMatch(/Failed to fetch|fetch failed/i);
    });

    test('reports JSON error payloads from non-2xx responses', async () => {
        // Given - a backend that said no, and said why
        await using _ = await intercept(
            http.post(primaryUrl),
            http.error(500, { error: 'Internal server error' }),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toStrictEqual(['Internal server error']);
    });

    test('uses fallbackUrl when the primary URL fails', async () => {
        // Given - a dead primary and a fallback that streams
        await using _ = await intercept(
            defineContracts(
                { request: http.post(primaryUrl), response: http.unreachable() },
                {
                    request: http.post(fallbackUrl),
                    response: http.sse([
                        {
                            data: {
                                message: { content: [{ text: 'fallback response', type: 'text' }] },
                                type: 'assistant',
                            },
                        },
                        done,
                    ]),
                },
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, fallbackUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.dones).toHaveLength(1);
        expect(cb.blocks.flat()).toContainEqual({ type: 'text', content: 'fallback response' });
    });

    test('reports the primary error when primary and fallback both fail', async () => {
        // Given - neither URL answers
        await using _ = await intercept(
            defineContracts(
                { request: http.post(primaryUrl), response: http.unreachable() },
                { request: http.post(fallbackUrl), response: http.unreachable() },
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, fallbackUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors[0]).toMatch(/Failed to fetch|fetch failed/i);
    });

    test('parses thinking and result error events', async () => {
        // Given - a thought, then a result event that carries a failure
        await using _ = await intercept(
            http.post(primaryUrl),
            http.sse([
                {
                    data: {
                        message: { content: [{ text: 'Let me think', type: 'thinking' }] },
                        type: 'assistant',
                    },
                },
                {
                    data: { result: 'Rate limit exceeded', subtype: 'error', type: 'result' },
                },
                done,
            ]),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.blocks.flat()).toContainEqual({ type: 'thinking', content: 'Let me think' });
        expect(cb.blocks.flat()).toContainEqual({ type: 'error', content: 'Rate limit exceeded' });
    });
});
