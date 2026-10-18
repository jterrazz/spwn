import { once } from 'node:events';
import type { Server } from 'node:http';
import { createServer } from 'node:net';
import { describe, expect, test, vi } from 'vitest';

import { Tool } from '../tool.ts';

const demo = (): Tool =>
    new Tool({ name: 'catalog-demo', title: 'Catalog Demo', version: '1.2.3' }).method('publish', {
        description: 'Publish a post',
        handler: ({ args }) => ({ args }),
        schema: { properties: { text: { type: 'string' } }, type: 'object' },
    });

// hazard: `run` sets process.exitCode, and the test process would exit with it, so it is read, then reset
async function captured(
    stream: NodeJS.WriteStream,
    run: () => Promise<unknown>,
): Promise<{ code: typeof process.exitCode; text: string }> {
    const chunks: string[] = [];
    const spy = vi.spyOn(stream, 'write').mockImplementation((chunk) => {
        chunks.push(String(chunk));
        return true;
    });
    await run();
    spy.mockRestore();
    const code = process.exitCode;
    process.exitCode = undefined;
    return { code, text: chunks.join('') };
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

async function exchange(port: number, path: string, body?: string) {
    const response = await fetch(
        `http://127.0.0.1:${port}${path}`,
        body === undefined ? undefined : { body, method: 'POST' },
    );
    return {
        body: await response.text(),
        status: response.status,
        type: response.headers.get('content-type'),
    };
}

function added(signal: NodeJS.Signals, before: readonly unknown[]): (() => void)[] {
    return process
        .listeners(signal)
        .filter((listener) => !before.includes(listener))
        .map((listener) => () => {
            Reflect.apply(listener, process, [signal]);
        });
}

const LIST = { id: 1, jsonrpc: '2.0', method: 'tools/list' };

function answered(status: number, body: unknown) {
    return { body: JSON.stringify(body), status, type: 'application/json' };
}

async function served() {
    const port = await freePort();
    vi.stubEnv('GATE_TOOL_PORT', String(port));
    const signals = { SIGINT: process.listeners('SIGINT'), SIGTERM: process.listeners('SIGTERM') };
    let server: Server | undefined;
    const { text } = await captured(process.stderr, async () => {
        server = await demo().run(['mcp-serve']);
        await once(server ?? probeFailed(), 'listening');
    });
    return { announced: text, port, server: server ?? probeFailed(), signals };
}

describe('Tool', () => {
    test('mcp-manifest prints the tool and its methods', async () => {
        // Given - a tool with one method described, and one registered with its handler alone
        const tool = demo().method('ping', { handler: () => 'pong' });
        const output = await captured(process.stdout, async () => await tool.run(['mcp-manifest']));

        // Then - the manifest the gate reads, whole
        expect(JSON.parse(output.text)).toStrictEqual({
            methods: [
                {
                    description: 'Publish a post',
                    inputSchema: { properties: { text: { type: 'string' } }, type: 'object' },
                    name: 'publish',
                },
                { description: '', inputSchema: { type: 'object' }, name: 'ping' },
            ],
            name: 'catalog-demo',
            title: 'Catalog Demo',
            version: '1.2.3',
        });
    });

    test('a method name runs that method once with the flags as its arguments', async () => {
        // Given - the method called from a host script
        const output = await captured(
            process.stdout,
            async () => await demo().run(['publish', '--text', 'hello', '--dry-run']),
        );

        // Then - its result, as one line of JSON
        expect(output).toStrictEqual({
            code: undefined,
            text: '{"args":{"text":"hello","dry-run":true}}\n',
        });
    });

    test('an unknown subcommand prints the usage and exits 2', async () => {
        // Given - a name the tool does not register
        const output = await captured(process.stderr, async () => await demo().run(['delete']));

        // Then - every subcommand, the tool's own after the two the gate uses
        expect(output).toStrictEqual({
            code: 2,
            text: `usage: catalog-demo <subcommand>

subcommands:
  mcp-manifest                     print the JSON tool list (gate uses this)
  mcp-serve                        run the MCP HTTP server (gate spawns this)
  publish                          Publish a post
`,
        });
    });

    test.each(['', '0', 'eighty'])('mcp-serve on port %j fails and exits 4', async (port) => {
        // Given - no port the gate could have assigned
        vi.stubEnv('GATE_TOOL_PORT', port);
        const output = await captured(process.stderr, async () => await demo().run(['mcp-serve']));

        // Then - the error and its stack
        expect(output.code).toBe(4);
        expect(output.text).toMatch(
            /^tool error: Error: mcp-serve: GATE_TOOL_PORT not set\n {4}at /u,
        );
    });

    test('mcp-serve listens on the loopback, says so, and stops on SIGINT or SIGTERM', async () => {
        // Given - the tool served on the port the gate assigned
        const { announced, port, server, signals } = await served();

        // Then - the address, the announcement, and one stop per signal
        const onTerm = added('SIGTERM', signals.SIGTERM);
        const onInt = added('SIGINT', signals.SIGINT);
        expect({
            address: server.address(),
            announced,
            stops: { SIGINT: onInt.length, SIGTERM: onTerm.length },
        }).toStrictEqual({
            address: { address: '127.0.0.1', family: 'IPv4', port },
            announced: `tool catalog-demo MCP server listening on 127.0.0.1:${port}\n`,
            stops: { SIGINT: 1, SIGTERM: 1 },
        });
        onInt[0]?.();
        await once(server, 'close');
        onTerm[0]?.();
    });

    test('mcp-serve answers MCP on POST, its health on GET /healthz, and refuses the rest', async () => {
        // Given - the tool served on the port the gate assigned
        const { port, server } = await served();

        // When - the gate lists the tools, asks the health, and sends what the endpoint refuses
        const exchanges = {
            empty: await exchange(port, '/mcp/', ''),
            health: await exchange(port, '/healthz?probe=1'),
            initialized: await exchange(
                port,
                '/mcp/',
                JSON.stringify({ jsonrpc: '2.0', method: 'notifications/initialized' }),
            ),
            listed: await exchange(port, '/healthz', JSON.stringify(LIST)),
            refused: await exchange(port, '/mcp/'),
        };
        server.close();

        // Then - every answer, whole
        expect(exchanges).toStrictEqual({
            empty: answered(400, {
                error: { code: -32_700, message: 'Unexpected end of JSON input' },
                jsonrpc: '2.0',
            }),
            health: answered(200, { methods: 1, name: 'catalog-demo', ok: true }),
            initialized: { body: '', status: 202, type: null },
            listed: answered(200, {
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
            }),
            refused: answered(405, { error: 'POST only' }),
        });
    });

    test('a method name is registered once', () => {
        // Given - the demo tool, which registers publish

        // Then - a second publish is refused
        expect(() => demo().method('publish', { handler: () => undefined })).toThrow(
            'method publish already registered',
        );
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
