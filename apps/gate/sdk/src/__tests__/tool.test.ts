import { once } from 'node:events';
import { createServer } from 'node:net';
import { describe, expect, test, vi } from 'vitest';

import { Tool } from '../tool.ts';

const demo = (): Tool =>
    new Tool({ name: 'catalog-demo', title: 'Catalog Demo', version: '1.2.3' }).method('publish', {
        description: 'Publish a post',
        handler: ({ args }) => ({ args }),
        schema: { properties: { text: { type: 'string' } }, type: 'object' },
    });

async function written(run: () => Promise<unknown>): Promise<string> {
    const chunks: string[] = [];
    const spy = vi.spyOn(process.stdout, 'write').mockImplementation((chunk) => {
        chunks.push(String(chunk));
        return true;
    });
    await run();
    spy.mockRestore();
    return chunks.join('');
}

async function freePort(): Promise<number> {
    const probe = createServer();
    probe.listen(0, '127.0.0.1');
    await once(probe, 'listening');
    const address = probe.address();
    probe.close();
    await once(probe, 'close');
    return typeof address === 'object' && address !== null ? address.port : 0;
}

function probeFailed(): never {
    throw new Error('mcp-serve returned no server');
}

describe('Tool', () => {
    test('mcp-manifest prints the tool and its methods', async () => {
        // Given - a tool with one method
        const output = await written(async () => await demo().run(['mcp-manifest']));

        // Then - the manifest the gate reads, whole
        expect(JSON.parse(output)).toStrictEqual({
            methods: [
                {
                    description: 'Publish a post',
                    inputSchema: { properties: { text: { type: 'string' } }, type: 'object' },
                    name: 'publish',
                },
            ],
            name: 'catalog-demo',
            title: 'Catalog Demo',
            version: '1.2.3',
        });
    });

    test('a method name runs that method once with the flags as its arguments', async () => {
        // Given - the method called from a host script
        const output = await written(
            async () => await demo().run(['publish', '--text', 'hello', '--dry-run']),
        );

        // Then - its result, as one line of JSON
        expect(output).toBe('{"args":{"text":"hello","dry-run":true}}\n');
    });

    test('mcp-serve answers MCP over HTTP on the port the gate assigns, and its health', async () => {
        // Given - the tool served on a free port
        const port = await freePort();
        vi.stubEnv('GATE_TOOL_PORT', String(port));
        const server = await demo().run(['mcp-serve']);
        if (server?.listening !== true) {
            await once(server ?? probeFailed(), 'listening');
        }

        // When - the gate lists the tools, asks the health, and sends a GET
        const listed = await fetch(`http://127.0.0.1:${port}/mcp/`, {
            body: JSON.stringify({ id: 1, jsonrpc: '2.0', method: 'tools/list' }),
            method: 'POST',
        });
        const health = await fetch(`http://127.0.0.1:${port}/healthz`);
        const refused = await fetch(`http://127.0.0.1:${port}/mcp/`);
        server?.close();

        // Then - the listing, the health and the refusal, whole
        expect({
            health: { body: (await health.json()) as unknown, status: health.status },
            listed: { body: (await listed.json()) as unknown, status: listed.status },
            refused: { body: (await refused.json()) as unknown, status: refused.status },
        }).toStrictEqual({
            health: { body: { methods: 1, name: 'catalog-demo', ok: true }, status: 200 },
            listed: {
                body: {
                    id: 1,
                    jsonrpc: '2.0',
                    result: {
                        tools: [
                            {
                                description: 'Publish a post',
                                inputSchema: {
                                    properties: { text: { type: 'string' } },
                                    type: 'object',
                                },
                                name: 'publish',
                            },
                        ],
                    },
                },
                status: 200,
            },
            refused: { body: { error: 'POST only' }, status: 405 },
        });
    });

    test('a tool with no name is refused, naming where the name comes from', () => {
        // Given - no name in the options and none in the environment
        vi.stubEnv('GATE_TOOL_NAME', '');

        // Then - the constructor refuses
        expect(() => new Tool()).toThrow(
            'Tool: name required (constructor opts or GATE_TOOL_NAME env)',
        );
    });
});
