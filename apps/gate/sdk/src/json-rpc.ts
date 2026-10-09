import { fieldOf } from './json.ts';
import type { Json } from './json.ts';

/** A method a tool registers: what it does, the JSON schema of its arguments, and the handler that answers it. */
export type Method = {
    description: string;
    handler: (call: { args: Record<string, unknown> }) => unknown;
    name: string;
    schema: Json;
};

export type Identity = { name: string; title: string; version: string };

/** The answer of one JSON-RPC exchange: the HTTP status and the body. */
export type Answer = { body: Json; status: number };

const PROTOCOL = '2025-06-18';

const ok = (id: Json, result: Json): Answer => ({
    body: { id, jsonrpc: '2.0', result },
    status: 200,
});

const failed = (id: Json, code: number, message: string): Answer => ({
    body: { error: { code, message }, id, jsonrpc: '2.0' },
    status: 200,
});

export const toolsOf = (methods: Iterable<Method>): Json[] =>
    [...methods].map((method) => ({
        description: method.description,
        inputSchema: method.schema,
        name: method.name,
    }));

async function called(
    id: Json,
    method: Method | undefined,
    name: string,
    args: Json,
): Promise<Answer> {
    if (method === undefined) {
        return failed(id, -32_601, `tool ${name} not found`);
    }
    try {
        const result = await method.handler({
            // invariant: spreading null gives `{}`, so null needs no branch; a string would spread into its characters
            args: typeof args === 'object' ? { ...args } : {},
        });
        const text = typeof result === 'string' ? result : JSON.stringify(result);
        return ok(id, { content: [{ text, type: 'text' }] });
    } catch (error) {
        return ok(id, {
            content: [
                { text: error instanceof Error ? error.message : String(error), type: 'text' },
            ],
            isError: true,
        });
    }
}

/** The answer to one MCP request a tool receives over HTTP. */
export async function answer(
    request: Json,
    identity: Identity,
    methods: ReadonlyMap<string, Method>,
): Promise<Answer> {
    const id = fieldOf(request, 'id');
    if (fieldOf(request, 'jsonrpc') !== '2.0') {
        return failed(id, -32_600, 'jsonrpc must be 2.0');
    }
    switch (fieldOf(request, 'method')) {
        case 'initialize': {
            return ok(id, {
                capabilities: { tools: {} },
                protocolVersion: PROTOCOL,
                serverInfo: identity,
            });
        }
        case 'notifications/initialized': {
            return { body: '', status: 202 };
        }
        case 'ping': {
            return ok(id, {});
        }
        case 'tools/list': {
            return ok(id, { tools: toolsOf(methods.values()) });
        }
        case 'tools/call': {
            const name = String(fieldOf(fieldOf(request, 'params'), 'name'));
            return await called(
                id,
                methods.get(name),
                name,
                fieldOf(fieldOf(request, 'params'), 'arguments'),
            );
        }
        default: {
            return failed(
                id,
                -32_601,
                `method ${String(fieldOf(request, 'method'))} not implemented`,
            );
        }
    }
}
