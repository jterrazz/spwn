import { createServer } from 'node:http';
import type { IncomingMessage, Server, ServerResponse } from 'node:http';

import { parseFlags } from './flags.ts';
import { answer, toolsOf } from './json-rpc.ts';
import type { Identity, Method } from './json-rpc.ts';
import type { Json } from './json.ts';

type Registration = Omit<Method, 'description' | 'name' | 'schema'> & {
    description?: string;
    schema?: Json;
};

const EXIT_USAGE = 2;
const EXIT_FAILED = 4;

function send(response: ServerResponse, status: number, body: Json): void {
    if (typeof body === 'string') {
        response.writeHead(status).end(body);
        return;
    }
    const data = JSON.stringify(body);
    response
        .writeHead(status, {
            'content-length': Buffer.byteLength(data),
            'content-type': 'application/json',
        })
        .end(data);
}

async function readJson(request: IncomingMessage): Promise<Json> {
    const chunks: Buffer[] = [];
    for await (const chunk of request) {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(String(chunk)));
    }
    return chunks.length === 0
        ? {}
        : (JSON.parse(Buffer.concat(chunks).toString('utf8')) as unknown);
}

/** A catalog tool: the methods it registers, served as MCP tools to the gate and as a CLI to a host script. */
export class Tool {
    readonly methods = new Map<string, Method>();
    private readonly identity: Identity;

    constructor(options: Partial<Identity> = {}) {
        const name = options.name ?? process.env.GATE_TOOL_NAME ?? '';
        if (name === '') {
            throw new Error('Tool: name required (constructor opts or GATE_TOOL_NAME env)');
        }
        this.identity = { name, title: options.title ?? name, version: options.version ?? '0.1.0' };
    }

    /** Register a method; its schema serves both the MCP listing and the CLI flags. */
    method(
        name: string,
        { description = '', handler, schema = { type: 'object' } }: Registration,
    ): this {
        if (this.methods.has(name)) {
            throw new Error(`method ${name} already registered`);
        }
        this.methods.set(name, { description, handler, name, schema });
        return this;
    }

    /** `mcp-manifest` prints the tool list, `mcp-serve` serves MCP on GATE_TOOL_PORT, a method name runs it once. */
    async run(argv: readonly string[] = process.argv.slice(2)): Promise<Server | undefined> {
        const [command = '', ...rest] = argv;
        try {
            if (command === 'mcp-manifest') {
                const manifest = { ...this.identity, methods: toolsOf(this.methods.values()) };
                process.stdout.write(`${JSON.stringify(manifest, null, 2)}\n`);
                return undefined;
            }
            if (command === 'mcp-serve') {
                return this.serve();
            }
            const method = this.methods.get(command);
            if (method === undefined) {
                process.stderr.write(this.usage());
                process.exitCode = EXIT_USAGE;
                return undefined;
            }
            const result: unknown = await method.handler({ args: parseFlags(rest) });
            process.stdout.write(`${JSON.stringify(result)}\n`);
        } catch (error) {
            process.stderr.write(
                `tool error: ${error instanceof Error ? (error.stack ?? error.message) : String(error)}\n`,
            );
            process.exitCode = EXIT_FAILED;
        }
        return undefined;
    }

    private usage(): string {
        const rows = [...this.methods.values()].map(
            (method) => `  ${method.name.padEnd(32)} ${method.description}\n`,
        );
        return [
            `usage: ${this.identity.name} <subcommand>\n\nsubcommands:\n`,
            '  mcp-manifest                     print the JSON tool list (gate uses this)\n',
            '  mcp-serve                        run the MCP HTTP server (gate spawns this)\n',
            ...rows,
        ].join('');
    }

    private serve(): Server {
        const port = Number.parseInt(process.env.GATE_TOOL_PORT ?? '0', 10);
        if (port === 0) {
            throw new Error('mcp-serve: GATE_TOOL_PORT not set');
        }
        const server = createServer((request, response) => {
            void this.respond(request, response);
        });
        server.listen(port, '127.0.0.1', () => {
            process.stderr.write(
                `tool ${this.identity.name} MCP server listening on 127.0.0.1:${port}\n`,
            );
        });
        const stop = (): void => {
            server.close();
        };
        process.once('SIGTERM', stop);
        process.once('SIGINT', stop);
        return server;
    }

    private async respond(request: IncomingMessage, response: ServerResponse): Promise<void> {
        if (request.method === 'GET' && request.url?.split('?')[0] === '/healthz') {
            send(response, 200, { methods: this.methods.size, name: this.identity.name, ok: true });
            return;
        }
        if (request.method !== 'POST') {
            send(response, 405, { error: 'POST only' });
            return;
        }
        let body: Json;
        try {
            body = await readJson(request);
        } catch (error) {
            send(response, 400, {
                error: {
                    code: -32_700,
                    message: error instanceof Error ? error.message : String(error),
                },
                jsonrpc: '2.0',
            });
            return;
        }
        const { body: reply, status } = await answer(body, this.identity, this.methods);
        send(response, status, reply);
    }
}
