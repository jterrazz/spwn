import { describe, expect, test } from 'vitest';

import { answer } from '../json-rpc.ts';
import type { Method } from '../json-rpc.ts';

const IDENTITY = { name: 'catalog-demo', title: 'Catalog Demo', version: '1.2.3' };

const count: Method = {
    description: 'Count items',
    handler: ({ args }) => ({ count: Number(args.limit ?? 0) }),
    name: 'count',
    schema: { properties: { limit: { type: 'number' } }, type: 'object' },
};
const broken: Method = {
    description: 'Always fails',
    handler: () => {
        throw new Error('the provider refused');
    },
    name: 'broken',
    schema: { type: 'object' },
};
const echo: Method = {
    description: 'Echo its arguments',
    handler: ({ args }) => args,
    name: 'echo',
    schema: { type: 'object' },
};
const greet: Method = {
    description: 'Greet',
    handler: () => 'hello',
    name: 'greet',
    schema: { type: 'object' },
};
const METHODS = new Map([
    ['broken', broken],
    ['count', count],
    ['echo', echo],
    ['greet', greet],
]);

describe('json-rpc', () => {
    test.each([
        {
            answer: {
                body: {
                    id: 1,
                    jsonrpc: '2.0',
                    result: {
                        capabilities: { tools: {} },
                        protocolVersion: '2025-06-18',
                        serverInfo: IDENTITY,
                    },
                },
                status: 200,
            },
            request: { id: 1, jsonrpc: '2.0', method: 'initialize' },
        },
        {
            answer: { body: '', status: 202 },
            request: { jsonrpc: '2.0', method: 'notifications/initialized' },
        },
        {
            answer: { body: { id: 2, jsonrpc: '2.0', result: {} }, status: 200 },
            request: { id: 2, jsonrpc: '2.0', method: 'ping' },
        },
        {
            answer: {
                body: {
                    id: 3,
                    jsonrpc: '2.0',
                    result: {
                        tools: [
                            {
                                description: 'Always fails',
                                inputSchema: { type: 'object' },
                                name: 'broken',
                            },
                            {
                                description: 'Count items',
                                inputSchema: count.schema,
                                name: 'count',
                            },
                            {
                                description: 'Echo its arguments',
                                inputSchema: { type: 'object' },
                                name: 'echo',
                            },
                            {
                                description: 'Greet',
                                inputSchema: { type: 'object' },
                                name: 'greet',
                            },
                        ],
                    },
                },
                status: 200,
            },
            request: { id: 3, jsonrpc: '2.0', method: 'tools/list' },
        },
        {
            answer: {
                body: {
                    id: 4,
                    jsonrpc: '2.0',
                    result: { content: [{ text: '{"count":4}', type: 'text' }] },
                },
                status: 200,
            },
            request: {
                id: 4,
                jsonrpc: '2.0',
                method: 'tools/call',
                params: { arguments: { limit: 4 }, name: 'count' },
            },
        },
        {
            answer: {
                body: {
                    id: 5,
                    jsonrpc: '2.0',
                    result: {
                        content: [{ text: 'the provider refused', type: 'text' }],
                        isError: true,
                    },
                },
                status: 200,
            },
            request: { id: 5, jsonrpc: '2.0', method: 'tools/call', params: { name: 'broken' } },
        },
        {
            answer: {
                body: {
                    id: 9,
                    jsonrpc: '2.0',
                    result: { content: [{ text: '{}', type: 'text' }] },
                },
                status: 200,
            },
            request: {
                id: 9,
                jsonrpc: '2.0',
                method: 'tools/call',
                params: { arguments: 'limit', name: 'echo' },
            },
        },
        {
            answer: {
                body: {
                    id: 10,
                    jsonrpc: '2.0',
                    result: { content: [{ text: 'hello', type: 'text' }] },
                },
                status: 200,
            },
            request: { id: 10, jsonrpc: '2.0', method: 'tools/call', params: { name: 'greet' } },
        },
        {
            answer: {
                body: {
                    error: { code: -32_601, message: 'tool missing not found' },
                    id: 6,
                    jsonrpc: '2.0',
                },
                status: 200,
            },
            request: { id: 6, jsonrpc: '2.0', method: 'tools/call', params: { name: 'missing' } },
        },
        {
            answer: {
                body: {
                    error: { code: -32_601, message: 'method resources/list not implemented' },
                    id: 7,
                    jsonrpc: '2.0',
                },
                status: 200,
            },
            request: { id: 7, jsonrpc: '2.0', method: 'resources/list' },
        },
        {
            answer: {
                body: {
                    error: { code: -32_600, message: 'jsonrpc must be 2.0' },
                    id: 8,
                    jsonrpc: '2.0',
                },
                status: 200,
            },
            request: { id: 8, jsonrpc: '1.0', method: 'ping' },
        },
    ])(
        'answers $request.method with the reply of the MCP contract',
        async ({ answer: expected, request }) => {
            // Given - one request as the gate forwards it

            // Then - the status and the whole body
            await expect(answer(request, IDENTITY, METHODS)).resolves.toStrictEqual(expected);
        },
    );
});
