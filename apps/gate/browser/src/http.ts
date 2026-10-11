import type { IncomingMessage, ServerResponse } from 'node:http';

/** A refusal the sidecar answers with its own status, and the code a tool can branch on. */
export class RefusedError extends Error {
    readonly code: null | string;
    readonly status: number;

    constructor(status: number, message: string, code: null | string = null) {
        super(message);
        this.name = 'RefusedError';
        this.code = code;
        this.status = status;
    }
}

export function send(response: ServerResponse, status: number, body: unknown): void {
    const data = typeof body === 'string' ? body : JSON.stringify(body);
    response.writeHead(status, {
        'Content-Length': Buffer.byteLength(data),
        'Content-Type': typeof body === 'string' ? 'text/plain' : 'application/json',
    });
    response.end(data);
}

/** The JSON object a request carries; an empty body reads as `{}`. */
export async function readJson(request: IncomingMessage): Promise<Record<string, unknown>> {
    const chunks: Buffer[] = [];
    for await (const chunk of request) {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(String(chunk)));
    }
    if (chunks.length === 0) {
        return {};
    }
    let parsed: unknown;
    try {
        parsed = JSON.parse(Buffer.concat(chunks).toString('utf8'));
    } catch (error) {
        throw new RefusedError(
            400,
            `bad json: ${error instanceof Error ? error.message : String(error)}`,
        );
    }
    // Stryker disable next-line ConditionalExpression: spreading null gives the same empty object, so a null check that always passes changes nothing
    return typeof parsed === 'object' && parsed !== null ? { ...parsed } : {};
}
