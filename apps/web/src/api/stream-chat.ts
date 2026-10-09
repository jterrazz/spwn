import { deduplicateBlocks, parseStreamEvent } from '../activity/blocks';
import type { ActivityBlock, StreamJsonEvent } from '../activity/blocks';
import { numberOf, textOf } from './fields';

export interface StreamMeta {
    cost?: number | undefined;
    duration?: number | undefined;
}

export interface StreamChatOptions {
    url: string;
    body: Record<string, unknown>;
    /** Receives the activity blocks as they arrive, each block once. */
    onBlocks: (blocks: ActivityBlock[]) => void;
    /** Receives the lines that are not stream events; without it they arrive as text blocks. */
    onText?: (text: string) => void;
    onDone: (meta: StreamMeta) => void;
    onError: (error: string) => void;
    signal?: AbortSignal;
    /** Milliseconds before the request is abandoned, when no signal is given. */
    timeout?: number;
}

const DATA_PREFIX = 'data: ';
const END_OF_STREAM = '[DONE]';
const REPLY_FIELDS = ['response', 'output', 'message', 'error'] as const;

function isStreamEvent(value: unknown): value is StreamJsonEvent {
    return (
        typeof value === 'object' &&
        value !== null &&
        typeof Reflect.get(value, 'type') === 'string'
    );
}

function eventOf(text: string): StreamJsonEvent | undefined {
    try {
        const value: unknown = JSON.parse(text);
        return isStreamEvent(value) ? value : undefined;
    } catch {
        return undefined;
    }
}

function replyOf(data: unknown): string {
    const field = REPLY_FIELDS.find((name) => textOf(data, name) !== '');
    return field === undefined ? 'No response' : textOf(data, field);
}

class StreamCollector {
    readonly meta: StreamMeta = {};
    private readonly seen: ActivityBlock[] = [];
    private readonly onBlocks: (blocks: ActivityBlock[]) => void;
    private readonly onText: ((text: string) => void) | undefined;

    constructor(
        onBlocks: (blocks: ActivityBlock[]) => void,
        onText: ((text: string) => void) | undefined,
    ) {
        this.onBlocks = onBlocks;
        this.onText = onText;
    }

    line(raw: string): void {
        const line = raw.trim();
        if (line.startsWith(DATA_PREFIX)) {
            const data = line.slice(DATA_PREFIX.length);
            if (data !== END_OF_STREAM) {
                this.payload(data, data);
            }
        } else if (line !== '' && !line.startsWith(':')) {
            this.payload(line, `${line}\n`);
        }
    }

    reply(text: string): void {
        this.onBlocks([{ content: text, type: 'text' }]);
    }

    private text(text: string): void {
        if (this.onText === undefined) {
            this.reply(text);
        } else {
            this.onText(text);
        }
    }

    private payload(json: string, verbatim: string): void {
        const event = eventOf(json);
        if (event === undefined) {
            this.text(verbatim);
            return;
        }
        this.record(event);
        const blocks = deduplicateBlocks(this.seen, parseStreamEvent(event) ?? []);
        if (blocks.length > 0) {
            this.seen.push(...blocks);
            this.onBlocks(blocks);
        }
    }

    private record(event: StreamJsonEvent): void {
        if (event.type === 'result') {
            this.meta.cost = numberOf(event, 'total_cost_usd');
            this.meta.duration = numberOf(event, 'duration_ms');
        } else if (event.type === 'turn.completed' && event.usage !== undefined) {
            this.meta.duration = undefined;
        }
    }
}

interface Stream {
    collector: StreamCollector;
    decoder: TextDecoder;
    reader: ReadableStreamDefaultReader<Uint8Array>;
}

async function drain(stream: Stream, pending = ''): Promise<void> {
    const { done, value } = await stream.reader.read();
    if (done) {
        return;
    }
    const lines = `${pending}${stream.decoder.decode(value, { stream: true })}`.split('\n');
    const rest = lines.pop() ?? '';
    for (const line of lines) {
        stream.collector.line(line);
    }
    await drain(stream, rest);
}

async function consume(response: Response, collector: StreamCollector): Promise<void> {
    if ((response.headers.get('content-type') ?? '').includes('application/json')) {
        const data: unknown = await response.json();
        collector.reply(replyOf(data));
        return;
    }
    if (response.body === null) {
        collector.reply(await response.text());
        return;
    }
    await drain({ collector, decoder: new TextDecoder(), reader: response.body.getReader() });
}

async function failureOf(response: Response): Promise<string> {
    try {
        const data: unknown = await response.json();
        const error = textOf(data, 'error');
        return error === '' ? `HTTP ${response.status}` : error;
    } catch {
        return `HTTP ${response.status}`;
    }
}

function messageOf(error: unknown, signal: AbortSignal | undefined): string {
    if (signal?.aborted === true) {
        return 'Request cancelled';
    }
    return error instanceof Error ? error.message : 'Failed to connect';
}

/** POST a chat message and read the reply as it streams: server-sent events, JSON lines, or one JSON body. */
export async function streamChat(options: StreamChatOptions): Promise<void> {
    const { body, onBlocks, onDone, onError, onText, signal, timeout = 120_000, url } = options;
    const collector = new StreamCollector(onBlocks, onText);
    try {
        const response = await fetch(url, {
            body: JSON.stringify(body),
            headers: { 'Content-Type': 'application/json' },
            method: 'POST',
            signal: signal ?? AbortSignal.timeout(timeout),
        });
        if (!response.ok) {
            throw new Error(await failureOf(response));
        }
        await consume(response, collector);
        onDone(collector.meta);
    } catch (error) {
        onError(messageOf(error, signal));
    }
}
