import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { afterAll, afterEach, beforeAll, describe, expect, test } from 'vitest';

import { streamChat } from '../stream-chat';

const primaryUrl = 'http://spwn.test/api/worlds/w-1/talk';
const fallbackUrl = 'http://fallback.test/api/architect/talk';

const server = setupServer();

beforeAll(() => {
    server.listen({ onUnhandledRequest: 'error' });
});

afterEach(() => {
    server.resetHandlers();
});

afterAll(() => {
    server.close();
});

function stream(chunks: string[]): ReadableStream<Uint8Array> {
    const encoder = new TextEncoder();
    return new ReadableStream({
        start(controller) {
            for (const chunk of chunks) {
                controller.enqueue(encoder.encode(chunk));
            }
            controller.close();
        },
    });
}

function sse(chunks: string[]): HttpResponse {
    return new HttpResponse(stream(chunks), {
        headers: { 'Content-Type': 'text/event-stream' },
    });
}

function callbacks() {
    const blocks: unknown[][] = [];
    const texts: string[] = [];
    const errors: string[] = [];
    const dones: Array<{ cost?: number; duration?: number }> = [];

    return {
        blocks,
        texts,
        errors,
        dones,
        onBlocks: (value: unknown[]) => blocks.push(value),
        onText: (value: string) => texts.push(value),
        onDone: (value: { cost?: number; duration?: number }) => dones.push(value),
        onError: (value: string) => errors.push(value),
    };
}

describe('streamChat', () => {
    test('parses assistant text from an SSE response', async () => {
        server.use(
            http.post(primaryUrl, () =>
                sse([
                    'data: {"type":"assistant","message":{"content":[{"type":"text","text":"hello world"}]}}\n\n',
                    'data: [DONE]\n\n',
                ]),
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.dones).toHaveLength(1);
        expect(cb.blocks.flat()).toContainEqual({ type: 'text', content: 'hello world' });
    });

    test('parses tool_use blocks from an SSE response', async () => {
        server.use(
            http.post(primaryUrl, () =>
                sse([
                    'data: {"type":"assistant","message":{"content":[{"type":"tool_use","name":"bash","id":"t1","input":{"command":"ls"}}]}}\n\n',
                    'data: [DONE]\n\n',
                ]),
            ),
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
        server.use(
            http.post(primaryUrl, () =>
                sse([
                    'data: {"type":"assistant","message":{"content":[{"type":"text","text":"done"}]}}\n\n',
                    'data: {"type":"result","subtype":"success","total_cost_usd":0.05,"duration_ms":1234}\n\n',
                    'data: [DONE]\n\n',
                ]),
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.dones).toEqual([{ cost: 0.05, duration: 1234 }]);
    });

    test('handles plain text streams', async () => {
        server.use(
            http.post(
                primaryUrl,
                () =>
                    new HttpResponse(stream(['Hello plain response.\n', 'Second line.\n']), {
                        headers: { 'Content-Type': 'text/plain' },
                    }),
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.dones).toHaveLength(1);
        expect(cb.texts).toContain('Hello plain response.\n');
    });

    test('handles JSON responses', async () => {
        server.use(
            http.post(primaryUrl, () => HttpResponse.json({ response: 'JSON fallback response' })),
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
        server.use(http.post(primaryUrl, () => HttpResponse.error()));

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.dones).toHaveLength(0);
        expect(cb.errors[0]).toMatch(/Failed to fetch|fetch failed/i);
    });

    test('reports JSON error payloads from non-2xx responses', async () => {
        server.use(
            http.post(primaryUrl, () =>
                HttpResponse.json({ error: 'Internal server error' }, { status: 500 }),
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toEqual(['Internal server error']);
    });

    test('uses fallbackUrl when the primary URL fails', async () => {
        server.use(
            http.post(primaryUrl, () => HttpResponse.error()),
            http.post(fallbackUrl, () =>
                sse([
                    'data: {"type":"assistant","message":{"content":[{"type":"text","text":"fallback response"}]}}\n\n',
                    'data: [DONE]\n\n',
                ]),
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, fallbackUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors).toHaveLength(0);
        expect(cb.dones).toHaveLength(1);
        expect(cb.blocks.flat()).toContainEqual({ type: 'text', content: 'fallback response' });
    });

    test('reports the primary error when primary and fallback both fail', async () => {
        server.use(
            http.post(primaryUrl, () => HttpResponse.error()),
            http.post(fallbackUrl, () => HttpResponse.error()),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, fallbackUrl, body: { message: 'hi' }, ...cb });

        expect(cb.errors[0]).toMatch(/Failed to fetch|fetch failed/i);
    });

    test('parses thinking and result error events', async () => {
        server.use(
            http.post(primaryUrl, () =>
                sse([
                    'data: {"type":"assistant","message":{"content":[{"type":"thinking","text":"Let me think"}]}}\n\n',
                    'data: {"type":"result","subtype":"error","result":"Rate limit exceeded"}\n\n',
                    'data: [DONE]\n\n',
                ]),
            ),
        );

        const cb = callbacks();
        await streamChat({ url: primaryUrl, body: { message: 'hi' }, ...cb });

        expect(cb.blocks.flat()).toContainEqual({ type: 'thinking', content: 'Let me think' });
        expect(cb.blocks.flat()).toContainEqual({ type: 'error', content: 'Rate limit exceeded' });
    });
});
